package handler

import (
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkmodel "github.com/zerok-ai/zk-utils-go/scenario/model"
	evaluator "github.com/zerok-ai/zk-utils-go/scenario/model/evaluators"
	zkredis "github.com/zerok-ai/zk-utils-go/storage/redis"
	"time"
)

var spanFilteringLogTag = "SpanFilteringHandler"

type SpanFilteringHandler struct {
	VersionedStore *zkredis.VersionedStore[zkmodel.Scenario]
	Cfg            *config.OtlpConfig
	ruleEvaluator  evaluator.RuleEvaluator
}

func NewSpanFilteringHandler(cfg *config.OtlpConfig) (*SpanFilteringHandler, error) {
	handler := SpanFilteringHandler{}
	store, err := zkredis.GetVersionedStore[zkmodel.Scenario](&cfg.Redis, common.RedisScenarioDbName, time.Duration(cfg.Scenario.SyncDuration))
	if err != nil {
		return nil, err
	}
	handler.VersionedStore = store
	handler.Cfg = cfg
	handler.ruleEvaluator = evaluator.NewRuleEvaluator()
	return &handler, nil
}

func (h *SpanFilteringHandler) findMatchingWorkLoadIds(spanDetails *model.SpanDetails) {
	scenarios := h.VersionedStore.GetAllValues()
	for _, scenario := range scenarios {
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
			}
		}
	}
}
