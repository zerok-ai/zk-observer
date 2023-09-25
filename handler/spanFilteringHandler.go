package handler

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkmodel "github.com/zerok-ai/zk-utils-go/scenario/model"
	evaluator "github.com/zerok-ai/zk-utils-go/scenario/model/evaluators"
	zkredis "github.com/zerok-ai/zk-utils-go/storage/redis"
	zktick "github.com/zerok-ai/zk-utils-go/ticker"
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
	redisHandler    *utils.RedisHandler
	workloadDetails sync.Map
	ctx             context.Context
	ticker          *zktick.TickerTask
	count           int
	startTime       time.Time
	pipeline        redis.Pipeliner
}

type WorkLoadTraceId struct {
	WorkLoadId string
	TraceId    string
}

func NewSpanFilteringHandler(cfg *config.OtlpConfig) (*SpanFilteringHandler, error) {
	rand.Seed(time.Now().UnixNano())
	handler := SpanFilteringHandler{}
	store, err := zkredis.GetVersionedStore[zkmodel.Scenario](&cfg.Redis, common.RedisScenarioDbName, time.Duration(cfg.Scenario.SyncDuration)*time.Second)
	if err != nil {
		return nil, err
	}
	redisHandler, err := utils.NewRedisHandler(&cfg.Redis, common.WorkloadSpanDbName)
	if err != nil {
		logger.Error(spanFilteringLogTag, "Error while creating workload redis handler:", err)
		return nil, err
	}

	handler.redisHandler = redisHandler
	handler.VersionedStore = store
	handler.Cfg = cfg
	handler.ruleEvaluator = evaluator.NewRuleEvaluator()
	handler.workloadDetails = sync.Map{}
	handler.ctx = context.Background()
	handler.pipeline = handler.redisHandler.RedisClient.Pipeline()
	return &handler, nil
}

func (h *SpanFilteringHandler) FilterSpans(spanDetails map[string]interface{}, traceId string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(spanFilteringLogTag, "Recovered from panic: ", r)
		}
	}()
	logger.Debug(spanFilteringLogTag, "Span details are: ", spanDetails)
	scenarios := h.VersionedStore.GetAllValues()
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
		for id, workload := range *workloads {
			rule := workload.Rule
			logger.Debug(spanFilteringLogTag, "Checking for workload id: ", id)
			value, err := h.ruleEvaluator.EvalRule(rule, spanDetails)
			if err != nil {
				continue
			}
			if value {
				logger.Debug(spanFilteringLogTag, "Span matched with scenario: ", scenario.Title, " workload id: ", id)
				currentTime := fmt.Sprintf("%v", time.Now().UnixNano())
				key := currentTime + "_" + h.getRandomNumber() + "_" + id
				h.workloadDetails.Store(key, WorkLoadTraceId{WorkLoadId: id, TraceId: traceId})
			}
		}
	}
	err := h.syncWorkloadsToRedis()
	if err != nil {
		logger.Error(spanFilteringLogTag, "Error while syncing workload data to redis pipeline ", err)
	}
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
		logger.Debug(spanFilteringLogTag, "Len of redis pipeline ", h.pipeline.Len())
		h.pipeline.SAdd(h.ctx, redisKey, traceId)
		h.pipeline.Expire(h.ctx, redisKey, time.Duration(h.Cfg.Workloads.Ttl)*time.Second)
		h.count++
		keysToDelete = append(keysToDelete, keyStr)
		return true
	})
	h.syncPipeline()
	// Delete the keys from the sync.Map after the iteration
	for _, key := range keysToDelete {
		h.workloadDetails.Delete(key)
	}
	return nil
}

func (h *SpanFilteringHandler) syncPipeline() {
	syncDuration := time.Duration(h.Cfg.Workloads.SyncDuration) * time.Millisecond
	if h.count > h.Cfg.Workloads.BatchSize || time.Since(h.startTime) >= syncDuration {
		_, err := h.pipeline.Exec(h.ctx)
		if err != nil {
			logger.Error(spanFilteringLogTag, "Error while syncing data to redis ", err)
			return
		}
		logger.Debug(spanFilteringLogTag, "Pipeline synchronized on batchsize/syncDuration")

		h.count = 0
		h.startTime = time.Now()
	}
}

func (h *SpanFilteringHandler) forceSync() {
	_, err := h.pipeline.Exec(h.ctx)
	if err != nil {
		logger.Error(spanFilteringLogTag, "Error while force syncing data to redis ", err)
		return
	}
}

func (h *SpanFilteringHandler) shutdown() {
	h.forceSync()
	err := h.redisHandler.CloseConnection()
	if err != nil {
		logger.Error(spanFilteringLogTag, "Error while closing redis conn.")
		return
	}
}

func (h *SpanFilteringHandler) getCurrentSuffix() (string, error) {

	currentTime := time.Now()
	timeFormatted := currentTime.Format("15:04")

	timeParts := strings.Split(timeFormatted, ":")
	minutesPart := timeParts[1]

	minutes, err := strconv.Atoi(minutesPart)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}
	suffix := minutes / 5
	return fmt.Sprintf("%v", suffix), nil
}

func (h *SpanFilteringHandler) getRandomNumber() string {
	randomNumber := rand.Intn(10000)
	return fmt.Sprintf("%v", randomNumber)
}
