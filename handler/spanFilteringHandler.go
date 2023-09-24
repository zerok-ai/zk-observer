package handler

import (
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkmodel "github.com/zerok-ai/zk-utils-go/scenario/model"
	evaluator "github.com/zerok-ai/zk-utils-go/scenario/model/evaluators"
	zkredis "github.com/zerok-ai/zk-utils-go/storage/redis"
	"math/rand"
	"time"
)

var spanFilteringLogTag = "SpanFilteringHandler"

type SpanFilteringHandler struct {
	VersionedStore *zkredis.VersionedStore[zkmodel.Scenario]
	Cfg            *config.OtlpConfig
	ruleEvaluator  evaluator.RuleEvaluator
	redisHandler   *utils.RedisHandler
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
	return &handler, nil
}

func (h *SpanFilteringHandler) FilterSpans(spanDetails *model.SpanDetails, traceId string) {
	filteredWorkloadIds := []string{}
	scenarios := h.VersionedStore.GetAllValues()
	//logger.Debug(spanFilteringLogTag, "Reached FilterSpans method.")
	for _, scenario := range scenarios {
		//logger.Debug(spanFilteringLogTag, "Checking for scenario ", scenario.Title)
		//Getting workloads and iterate over them
		workloads := scenario.Workloads
		for id, workload := range *workloads {
			rule := workload.Rule
			value, err := h.ruleEvaluator.EvalRule(rule, spanDetails.Attributes)
			if err != nil {
				continue
			}
			if value {
				logger.Debug(spanFilteringLogTag, "Span matched with scenario: ", scenario.Title, " workload id: ", id)
				filteredWorkloadIds = append(filteredWorkloadIds, id)
			}
		}
	}
	h.saveWorkloadsToRedis(filteredWorkloadIds, traceId)
}

func (h *SpanFilteringHandler) saveWorkloadsToRedis(workloadIds []string, traceId string) {
	for _, workloadId := range workloadIds {
		key := workloadId + "_" + h.getCurrentSuffix()
		//TODO: Confirm with avin what is the format here.
		logger.Debug(spanFilteringLogTag, "Setting value for key: ", key, " workloadId ", workloadId)
	}
}

func (h *SpanFilteringHandler) getCurrentSuffix() string {
	//TODO: Confirm with avin, if this is okay for getting the key.
	randomNumber := rand.Intn(11)
	return fmt.Sprintf("%v", randomNumber)
}
