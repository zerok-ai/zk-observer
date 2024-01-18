package redis

import (
	"context"
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	zkUtilsCommonModel "github.com/zerok-ai/zk-utils-go/common"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkmodel "github.com/zerok-ai/zk-utils-go/scenario/model"
	evaluator "github.com/zerok-ai/zk-utils-go/scenario/model/evaluators"
	"github.com/zerok-ai/zk-utils-go/scenario/model/evaluators/functions"
	zkredis "github.com/zerok-ai/zk-utils-go/storage/redis"
	"github.com/zerok-ai/zk-utils-go/storage/redis/clientDBNames"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
	"k8s.io/utils/strings/slices"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var spanFilteringLogTag = "SpanFilteringHandler"

type SpanFilteringHandler struct {
	VersionedStore    *zkredis.VersionedStore[zkmodel.Scenario]
	Cfg               *config.OtlpConfig
	ruleEvaluator     evaluator.RuleEvaluator
	redisHandler      *RedisHandler
	workloadDetails   sync.Map
	ctx               context.Context
	executorAttrStore *stores.ExecutorAttrStore
	podDetailsStore   *stores.LocalCacheHSetStore
}

type WorkLoadTraceId struct {
	WorkLoadId string
	TraceId    string
}

type WorkloadIdList []string

func NewSpanFilteringHandler(cfg *config.OtlpConfig, executorAttrStore *stores.ExecutorAttrStore, podDetailsStore *stores.LocalCacheHSetStore) (*SpanFilteringHandler, error) {
	rand.Seed(time.Now().UnixNano())
	store, err := zkredis.GetVersionedStore[zkmodel.Scenario](&cfg.Redis, clientDBNames.ScenariosDBName, time.Duration(cfg.Scenario.SyncDuration)*time.Second)
	if err != nil {
		return nil, err
	}
	redisHandler, err := NewRedisHandler(&cfg.Redis, clientDBNames.FilteredTracesDBName, cfg.Workloads.SyncDuration, cfg.Workloads.BatchSize, resourceLogTag)
	if err != nil {
		logger.Error(resourceLogTag, "Error while creating resource redis handler:", err)
		return nil, err
	}

	handler := SpanFilteringHandler{
		VersionedStore:    store,
		Cfg:               cfg,
		ruleEvaluator:     *evaluator.NewRuleEvaluator(executorAttrStore, podDetailsStore),
		workloadDetails:   sync.Map{},
		ctx:               context.Background(),
		redisHandler:      redisHandler,
		executorAttrStore: executorAttrStore,
		podDetailsStore:   podDetailsStore,
	}

	return &handler, nil
}

func (h *SpanFilteringHandler) FilterSpans(traceId string, spanDetailsMap map[string]interface{}) (WorkloadIdList, zkUtilsCommonModel.GroupByMap) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(spanFilteringLogTag, "FilterSpans: Recovered from panic: ", r)
		}
	}()
	scenarios := h.VersionedStore.GetAllValues()
	var satisfiedWorkLoadIds WorkloadIdList
	var groupByMap zkUtilsCommonModel.GroupByMap
	for _, scenario := range scenarios {
		if scenario == nil {
			logger.Info(spanFilteringLogTag, "No scenario found")
			continue
		}
		processedWorkloadIds := h.processScenarioWorkloads(scenario, traceId, spanDetailsMap)
		if len(processedWorkloadIds) > 0 {
			if satisfiedWorkLoadIds == nil {
				satisfiedWorkLoadIds = make(WorkloadIdList, 0)
			}
			satisfiedWorkLoadIds = append(satisfiedWorkLoadIds, processedWorkloadIds...)
			if groupByValues, hasData := h.processGroupBy(scenario, spanDetailsMap, satisfiedWorkLoadIds); hasData && len(groupByValues) != 0 {
				if groupByMap == nil {
					groupByMap = make(zkUtilsCommonModel.GroupByMap)
				}
				groupByMap[zkUtilsCommonModel.ScenarioId(scenario.Id)] = groupByValues
			}
		}
	}
	err := h.syncWorkloadsToRedis()
	if err != nil {
		logger.Error(spanFilteringLogTag, "Error while syncing workload data to redis pipeline ", err)
	}
	return satisfiedWorkLoadIds, groupByMap
}

func (h *SpanFilteringHandler) processGroupBy(scenario *zkmodel.Scenario, spanDetailsMap map[string]interface{}, satisfiedWorkLoadIds WorkloadIdList) (zkUtilsCommonModel.GroupByValues, bool) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(spanFilteringLogTag, "processGroupBy: Recovered from panic: ", r)
		}
	}()

	var hasValueForScenario = false
	var groupByValues = make(zkUtilsCommonModel.GroupByValues, len(scenario.GroupBy))
	ff := functions.NewFunctionFactory(h.podDetailsStore, h.executorAttrStore)

	for idx, groupByItem := range scenario.GroupBy {
		protocol := getProtocolForWorkloadId(groupByItem.WorkloadId, scenario)
		attribKey := utils.GenerateAttribStoreKey(spanDetailsMap, protocol)
		// check if groupByItem.WorkloadId is present in satisfiedWorkLoadIds
		if slices.Contains(satisfiedWorkLoadIds, groupByItem.WorkloadId) {
			//Getting title and hash from executor attributes
			titleVal, _ := ff.EvaluateString(groupByItem.Title, spanDetailsMap, &attribKey)
			hashVal, _ := ff.EvaluateString(groupByItem.Hash, spanDetailsMap, &attribKey)
			groupByValues[idx] = &zkUtilsCommonModel.GroupByValueItem{
				WorkloadId: groupByItem.WorkloadId,
				Title:      fmt.Sprintf("%v", titleVal),
				Hash:       fmt.Sprintf("%v", hashVal),
			}
			hasValueForScenario = true
		} else {
			logger.Debug(spanFilteringLogTag, "WorkloadId ", groupByItem.WorkloadId, " not present in satisfiedWorkLoadIds")
			groupByValues[idx] = nil
		}
	}
	return groupByValues, hasValueForScenario
}

func getProtocolForWorkloadId(workloadID string, scenario *zkmodel.Scenario) zkmodel.ProtocolName {
	workload, ok := (*scenario.Workloads)[workloadID]
	if !ok {
		return zkmodel.ProtocolHTTP
	}
	return workload.Protocol
}

func (h *SpanFilteringHandler) IsSpanToBeEvaluated(workload zkmodel.Workload, spanDetailsMap map[string]interface{}) bool {
	if workload.Executor != zkmodel.ExecutorOTel {
		logger.Debug(spanFilteringLogTag, "Workload executor is not OTel")
		return false
	}

	workloadServiceName := workload.Service
	// if workloadServiceName contains "/" then it is a otel service name, else evaluate as k8s namespace/workload name
	if !strings.Contains(workloadServiceName, "/") {
		return h.isSpanToBeEvaluatedForOTelService(workload, spanDetailsMap)
	} else {
		return h.isSpanToBeEvaluatedForK8sWorkload(workload, spanDetailsMap)
	}
}

func (h *SpanFilteringHandler) isSpanToBeEvaluatedForK8sWorkload(workload zkmodel.Workload, spanDetailsMap map[string]interface{}) bool {
	scenarioWorkloadNs, scenarioWorkloadDeplName, err := workload.GetNamespaceAndWorkloadName()
	if err != nil {
		logger.Debug(spanFilteringLogTag, "Error while getting namespace and workload name for workload service: ", workload.Service, " error: ", err)
		return false
	}

	resourceAttributes, ok := spanDetailsMap[common.OTelResourceAttrKey]
	if !ok || resourceAttributes == nil {
		logger.Warn(spanFilteringLogTag, "Resource attributes not found in spanDetailsMap")
		return true
	}
	resourceAttrMap := resourceAttributes.(map[string]interface{})
	//var k8sNamespace, k8sDeployment string
	k8sNamespace, nsOk := resourceAttrMap[common.OTelResourceAttrNamespaceKey]
	if !nsOk || k8sNamespace == "" {
		logger.Warn(spanFilteringLogTag, "Namespace not found in resourceAttrMap, using namespace='*'. "+
			"Please set k8s.namespace.name value in OTEL_RESOURCE_ATTRIBUTES env variable.")
		k8sNamespace = common.ScenarioWorkloadGenericNamespaceKey
	}
	k8sDeployment, deplOk := resourceAttrMap[common.OTelResourceAttrDeploymentNameKey]
	if !deplOk || k8sDeployment == "" {
		logger.Warn(spanFilteringLogTag, "Deployment not found in resourceAttrMap, using deploymentName='*'. "+
			"Please set k8s.deployment.name value in OTEL_RESOURCE_ATTRIBUTES env variable.")
		k8sDeployment = common.ScenarioWorkloadGenericDeploymentKey
	}

	if scenarioWorkloadNs != common.ScenarioWorkloadGenericNamespaceKey && k8sNamespace != scenarioWorkloadNs {
		logger.Debug(spanFilteringLogTag, "NS::resourceAttrMap: ", k8sNamespace, "scenarioWorkload: ", scenarioWorkloadNs)
		logger.Info(spanFilteringLogTag, "Workload namespaces are not matching")
		return false
	}

	if scenarioWorkloadDeplName != common.ScenarioWorkloadGenericDeploymentKey && k8sDeployment != scenarioWorkloadDeplName {
		logger.Debug(spanFilteringLogTag, "NS::resourceAttrMap: ", k8sDeployment, "scenarioWorkload: ", scenarioWorkloadDeplName)
		logger.Info(spanFilteringLogTag, "Workload deployments are not matching")
		return false
	}
	return true
}

func (h *SpanFilteringHandler) isSpanToBeEvaluatedForOTelService(workload zkmodel.Workload, spanDetailsMap map[string]interface{}) bool {
	workloadServiceName := workload.Service
	spanAttributes, ok := spanDetailsMap[common.OTelSpanAttrKey]
	if !ok || spanAttributes == nil {
		logger.Warn(spanFilteringLogTag, "Resource attributes not found in spanDetailsMap")
		return true
	}
	spanAttrMap := spanAttributes.(map[string]interface{})
	spanServiceName, nsOk := spanAttrMap[common.OTelSpanAttrServiceNameKey]
	if !nsOk || spanServiceName == "" {
		logger.Warn(spanFilteringLogTag, "Service Name not found in resourceAttrMap, using service name='*'. "+
			"Please set OTEL_SERVICE_NAME or `service.name` key in OTEL_RESOURCE_ATTRIBUTES env variable.")
		spanServiceName = common.ScenarioWorkloadGenericServiceNameKey
	}

	if spanServiceName != common.ScenarioWorkloadGenericDeploymentKey && spanServiceName != workloadServiceName {
		logger.Debug(spanFilteringLogTag, "NS::spanAttrMap: ", spanServiceName, "scenarioWorkload: ", workloadServiceName)
		logger.Info(spanFilteringLogTag, "OTel service name is not matching")
		return false
	}
	return true
}

func (h *SpanFilteringHandler) processScenarioWorkloads(scenario *zkmodel.Scenario, traceId string, spanDetailsMap map[string]interface{}) WorkloadIdList {
	var satisfiedWorkLoadIds = make(WorkloadIdList, 0)
	//Getting workloads and iterate over them
	workloads := scenario.Workloads
	if workloads == nil {
		logger.Debug(spanFilteringLogTag, "No workloads found for scenario: ", scenario.Title)
		return satisfiedWorkLoadIds
	}
	for id, workload := range *workloads {
		// Check if span is to be evaluated for this workload
		if !h.IsSpanToBeEvaluated(workload, spanDetailsMap) {
			logger.Debug(spanFilteringLogTag, "Span not to be evaluated for workload: ", id)
			continue
		}

		rule := workload.Rule
		logger.Debug(spanFilteringLogTag, "Evaluating rule for scenario: ", scenario.Title, " workload id: ", id)
		attribKey := utils.GenerateAttribStoreKey(spanDetailsMap, workload.Protocol)
		value, err := h.ruleEvaluator.EvalRule(rule, attribKey, spanDetailsMap)
		if err != nil {
			logger.Info(spanFilteringLogTag, "Error while evaluating rule for scenario: ", scenario.Title, " workload id: ", id, " error: ", err)
			continue
		}
		if value {
			currentTime := fmt.Sprintf("%v", time.Now().UnixNano())
			key := currentTime + "_" + h.getRandomNumber() + "_" + id
			h.workloadDetails.Store(key, WorkLoadTraceId{WorkLoadId: id, TraceId: traceId})
			satisfiedWorkLoadIds = append(satisfiedWorkLoadIds, id)
		}
	}
	return satisfiedWorkLoadIds
}

func (h *SpanFilteringHandler) syncWorkloadsToRedis() error {

	if err := h.redisHandler.CheckRedisConnection(); err != nil {
		logger.Error(spanFilteringLogTag, "Error while checking redis conn ", err)
		return err
	}
	var keysToDelete []string
	h.workloadDetails.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		workLoadTraceId := value.(WorkLoadTraceId)

		workloadId := workLoadTraceId.WorkLoadId
		traceId := workLoadTraceId.TraceId

		redisKey := workloadId + "_latest"
		err := h.redisHandler.SAddPipeline(redisKey, traceId, time.Duration(h.Cfg.Workloads.Ttl)*time.Second)
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

func (h *SpanFilteringHandler) getRandomNumber() string {
	randomNumber := rand.Intn(10000)
	return fmt.Sprintf("%v", randomNumber)
}

func (h *SpanFilteringHandler) SyncPipeline() {
	h.redisHandler.SyncPipeline()
}
