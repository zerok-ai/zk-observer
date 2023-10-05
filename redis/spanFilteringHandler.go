package redis

import (
	"context"
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkmodel "github.com/zerok-ai/zk-utils-go/scenario/model"
	evaluator "github.com/zerok-ai/zk-utils-go/scenario/model/evaluators"
	zkredis "github.com/zerok-ai/zk-utils-go/storage/redis"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

var spanFilteringLogTag = "SpanFilteringHandler"

type SpanFilteringHandler struct {
	VersionedStore  *zkredis.VersionedStore[zkmodel.Scenario]
	Cfg             *config.OtlpConfig
	ruleEvaluator   evaluator.RuleEvaluator
	redisHandler    *RedisHandler
	workloadDetails sync.Map
	ctx             context.Context
}

type WorkLoadTraceId struct {
	WorkLoadId string
	TraceId    string
}

func NewSpanFilteringHandler(cfg *config.OtlpConfig) (*SpanFilteringHandler, error) {
	rand.Seed(time.Now().UnixNano())
	store, err := zkredis.GetVersionedStore[zkmodel.Scenario](&cfg.Redis, common.ScenariosDBName, time.Duration(cfg.Scenario.SyncDuration)*time.Second)
	if err != nil {
		return nil, err
	}

	redisHandler, err := NewRedisHandler(&cfg.Redis, common.FilteredTracesDBName, cfg.Workloads.SyncDuration, cfg.Workloads.BatchSize, resourceLogTag)
	if err != nil {
		logger.Error(resourceLogTag, "Error while creating resource redis handler:", err)
		return nil, err
	}

	handler := SpanFilteringHandler{
		VersionedStore:  store,
		Cfg:             cfg,
		ruleEvaluator:   evaluator.NewRuleEvaluator(cfg.Redis, zkmodel.ExecutorOTel, context.Background()),
		workloadDetails: sync.Map{},
		ctx:             context.Background(),
		redisHandler:    redisHandler,
	}

	return &handler, nil
}

func (h *SpanFilteringHandler) FilterSpans(spanDetails map[string]interface{}, traceId string) []string {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(spanFilteringLogTag, "Recovered from panic: ", r)
		}
	}()
	scenarios := h.VersionedStore.GetAllValues()
	satisfiedWorkLoadIds := make([]string, 0)
	for _, scenario := range scenarios {
		if scenario == nil {
			logger.Debug(spanFilteringLogTag, "No scenario found")
			continue
		}
		//Getting workloads and iterate over them
		workloads := scenario.Workloads
		if workloads == nil {
			logger.Debug(spanFilteringLogTag, "No workloads found for scenario: ", scenario.Title)
			continue
		}
		logger.Debug(spanFilteringLogTag, "Workloads found for scenario: ", scenario.Title, " workloads: ", workloads)
		for id, workload := range *workloads {
			if workload.Executor != zkmodel.ExecutorOTel {
				logger.Debug(spanFilteringLogTag, "Workload executor is not OTel")
				continue
			}
			rule := workload.Rule
			schemaVersion, ok := spanDetails["schema_version"]
			if !ok {
				schemaVersion = common.DefaultSchemaVersion
			}

			logger.Debug(spanFilteringLogTag, "Evaluating rule for scenario: ", scenario.Title, " workload id: ", id)
			value, err := h.ruleEvaluator.EvalRule(rule, schemaVersion.(string), workload.Protocol, spanDetails)
			if err != nil {
				logger.Error(spanFilteringLogTag, "Error while evaluating rule for scenario: ", scenario.Title, " workload id: ", id, " error: ", err)
				continue
			}
			if value {
				logger.Debug(spanFilteringLogTag, "Span matched with scenario: ", scenario.Title, " workload id: ", id)
				currentTime := fmt.Sprintf("%v", time.Now().UnixNano())
				key := currentTime + "_" + h.getRandomNumber() + "_" + id
				h.workloadDetails.Store(key, WorkLoadTraceId{WorkLoadId: id, TraceId: traceId})
				satisfiedWorkLoadIds = append(satisfiedWorkLoadIds, id)
			}
		}
	}
	err := h.syncWorkloadsToRedis()
	if err != nil {
		logger.Error(spanFilteringLogTag, "Error while syncing workload data to redis pipeline ", err)
	}
	return satisfiedWorkLoadIds
}

func (h *SpanFilteringHandler) syncWorkloadsToRedis() error {
	err := h.redisHandler.CheckRedisConnection()
	if err != nil {
		logger.Error(spanFilteringLogTag, "Error while checking redis conn ", err)
		return err
	}
	var keysToDelete []string
	h.workloadDetails.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		workLoadTraceId := value.(WorkLoadTraceId)

		workloadId := workLoadTraceId.WorkLoadId
		traceId := workLoadTraceId.TraceId

		suffix, err := h.getCurrentSuffix()
		if err != nil {
			logger.Error(spanFilteringLogTag, "Error while getting suffix for workloadId ", workloadId)
			return true
		}
		redisKey := workloadId + "_" + suffix
		logger.Debug(spanFilteringLogTag, "Setting value for key: ", redisKey, " workloadId ", workloadId)
		//logger.Debug(spanFilteringLogTag, "Len of redis pipeline ", h.pipeline.Len())
		err = h.redisHandler.SAddPipeline(redisKey, traceId, time.Duration(h.Cfg.Workloads.Ttl)*time.Second)
		if err != nil {
			logger.Error(spanFilteringLogTag, "Error while setting workload data: ", err)
			return true
		}
		keysToDelete = append(keysToDelete, keyStr)
		return true
	})
	// Delete the keys from the sync.Map after the iteration
	for _, key := range keysToDelete {
		h.workloadDetails.Delete(key)
	}
	return nil
}

func (h *SpanFilteringHandler) getCurrentSuffix() (string, error) {

	currentTime := time.Now()
	timeFormatted := currentTime.Format("15:04")

	timeParts := strings.Split(timeFormatted, ":")
	minutesPart := timeParts[1]

	minutes, err := strconv.Atoi(minutesPart)
	if err != nil {
		logger.Error(spanFilteringLogTag, "Error while converting minutes to int.", err)
		return "", err
	}
	suffix := minutes / 5
	return fmt.Sprintf("%v", suffix), nil
}

func (h *SpanFilteringHandler) getRandomNumber() string {
	randomNumber := rand.Intn(10000)
	return fmt.Sprintf("%v", randomNumber)
}

func (h *SpanFilteringHandler) SyncPipeline() {
	h.redisHandler.SyncPipeline()
}
