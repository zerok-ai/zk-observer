package handler

import (
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-otlp-receiver/redis"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/podDetails"
	ExecutorModel "github.com/zerok-ai/zk-utils-go/scenario/model"
	"github.com/zerok-ai/zk-utils-go/scenario/model/evaluators/cache"
	zkredis "github.com/zerok-ai/zk-utils-go/storage/redis"
	"github.com/zerok-ai/zk-utils-go/storage/redis/clientDBNames"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"io"
	"strings"
	"sync"
	"time"
)

var traceLogTag = "TraceHandler"
var delimiter = "__"
var DefaultNodeJsSchemaUrl = "https://opentelemetry.io/schemas/1.7.0"

type TraceHandler struct {
	traceStore             sync.Map
	traceStoreMutex        sync.Mutex
	traceRedisHandler      *redis.TraceRedisHandler
	exceptionHandler       *redis.ExceptionRedisHandler
	resourceDetailsHandler *redis.ResourceRedisHandler
	otlpConfig             *config.OtlpConfig
	spanFilteringHandler   *redis.SpanFilteringHandler
	factory                stores.StoreFactory
	workloadKeyHandler     *redis.WorkloadKeyHandler
}

func NewTraceHandler(config *config.OtlpConfig, factory stores.StoreFactory) (*TraceHandler, error) {
	handler := TraceHandler{}
	traceRedisHandler, err := redis.NewTracesRedisHandler(config)
	if err != nil {
		logger.Error(traceLogTag, "Error while creating redis handler:", err)
		return nil, err
	}

	exceptionHandler, err := redis.NewExceptionHandler(config)
	if err != nil {
		logger.Error(traceLogTag, "Error while creating exception handler:", err)
		return nil, err
	}

	resourceHandler, err := redis.NewResourceDetailsHandler(config)
	if err != nil {
		logger.Error(traceLogTag, "Error while creating resource details handler:", err)
		return nil, err
	}

	executorAttrStore := factory.GetExecutorAttrStore()
	podDetailsStore := factory.GetPodDetailsStore()

	handler.resourceDetailsHandler = resourceHandler
	handler.exceptionHandler = exceptionHandler
	handler.traceStore = sync.Map{}
	handler.traceRedisHandler = traceRedisHandler
	handler.otlpConfig = config
	handler.factory = factory

	scenarioStore, err := zkredis.GetVersionedStore[ExecutorModel.Scenario](&config.Redis, clientDBNames.ScenariosDBName, time.Duration(config.Scenario.SyncDuration)*time.Second)
	if err != nil {
		logger.Error(traceLogTag, "Error while scenarioDb store:", err)
		return nil, err
	}

	spanFilteringHandler, err := redis.NewSpanFilteringHandler(config, executorAttrStore, podDetailsStore, scenarioStore)
	if err != nil {
		logger.Error(traceLogTag, "Error while creating span filtering handler:", err)
		return nil, err
	}
	handler.spanFilteringHandler = spanFilteringHandler

	workLoadKeyHandler, err := redis.NewWorkloadKeyHandler(config, scenarioStore)
	if err != nil {
		logger.Error(traceLogTag, "Error while creating workLoadKeyHandler:", err)
		return nil, err
	}
	handler.workloadKeyHandler = workLoadKeyHandler

	return &handler, nil
}

func (th *TraceHandler) ServeHTTP(ctx iris.Context) {
	// Read the request body
	body, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		return
	}

	//logger.Debug(traceLogTag, string(body))

	// Unmarshal the protobuf message from the request body
	var traceData tracev1.TracesData
	err = proto.Unmarshal(body, &traceData)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		return
	}

	resourceSpans := traceData.ResourceSpans
	th.ProcessTraceData(resourceSpans)
	th.PushDataToRedis()

	// Respond to the client
	ctx.StatusCode(iris.StatusOK)
}

func (th *TraceHandler) PushDataToRedis() {
	// Iterate over the sync.Map and push the data to redis
	keysToDelete := th.pushSpansToRedisPipeline()

	// Delete the keys from the sync.Map after the iteration
	th.deleteFromTraceStore(keysToDelete)

	th.traceRedisHandler.SyncPipeline()
	th.exceptionHandler.SyncPipeline()
	th.resourceDetailsHandler.SyncPipeline()
	th.spanFilteringHandler.SyncPipeline()
}

func (th *TraceHandler) ProcessTraceData(resourceSpans []*tracev1.ResourceSpans) {
	var processedSpanCount = 0
	if len(resourceSpans) == 0 {
		logger.Info(traceLogTag, "No resources found in the call")
		return
	}
	for _, resourceSpan := range resourceSpans {
		schemaUrl := resourceSpan.SchemaUrl
		if len(schemaUrl) == 0 {
			schemaUrl = DefaultNodeJsSchemaUrl
		}
		schemaVersion := utils.GetSchemaVersion(schemaUrl)
		resourceAttrMap := map[string]interface{}{}
		if resourceSpan.Resource != nil {
			resourceAttrMap = utils.ConvertKVListToMap(resourceSpan.Resource.Attributes)
		}
		for _, scopeSpans := range resourceSpan.ScopeSpans {
			scopeAttrMap := map[string]interface{}{}
			if scopeSpans.Scope != nil {
				scopeAttrMap = utils.ConvertKVListToMap(scopeSpans.Scope.Attributes)
			}
			for _, span := range scopeSpans.Spans {
				processedSpanCount++
				traceId := hex.EncodeToString(span.TraceId)
				spanId := hex.EncodeToString(span.SpanId)
				logger.Debug(traceLogTag, "traceId", traceId, " , spanId", spanId, " , spanKind ", span.Kind, " ,parentSpanId ", hex.EncodeToString(span.ParentSpanId))
				if traceId == "" || spanId == "" {
					logger.Warn(traceLogTag, "TraceId or SpanId is empty for span ", spanId)
					continue
				}

				//TODO: Make this Async.
				//spanDetails := th.generateSpanDetails(span, schemaVersion, resourceAttrMap, scopeAttrMap)
				//workloadIds, groupBy := th.filterSpansOnZkSpanDetails(traceId, spanId, spanDetails)
				//spanDetails.WorkloadIdList = workloadIds
				//spanDetails.GroupBy = groupBy

				logger.Debug(traceLogTag, "span schema version:", schemaVersion)
				workloadIds, groupBy := th.filterSpansOnOTelSpanDetails(traceId, spanId, utils.ObjectToInterfaceMap(span), resourceAttrMap, scopeAttrMap)

				spanAttributes := utils.ConvertKVListToMap(span.Attributes)
				spanKind := model.NewFromOTelSpan(span.Kind)
				sourceIP, destIP := utils.GetSourceDestIPPair(spanKind, spanAttributes, resourceAttrMap)
				resourceIp := utils.GetResourceIp(spanKind, sourceIP, destIP)

				//Updating the spanDetails in traceStore.
				key := traceId + delimiter + spanId
				th.addToTraceStore(key, spanDetails)

				err := th.resourceDetailsHandler.SyncResourceData(resourceIp, resourceAttrMap)
				if err != nil {
					logger.Error(traceLogTag, "Error while saving resource data to redis for spanId ", spanId, " error is ", err)
				}
			}
		}
	}
	defer logger.InfoF(traceLogTag, "Processed %v spans", processedSpanCount)
}

// Execute rules on the span and populate the span details.
func (th *TraceHandler) filterSpansOnOTelSpanDetails(traceId string, spanId string, spanDetailsMap model.GenericMap, spanAttrMap model.GenericMap, resourceAttrMap model.GenericMap) ([]string, model.GroupByMap) {
	logger.Debug(traceLogTag, "Performing span filtering on OTel span ", spanId)
	workloadIds, groupBy := th.spanFilteringHandler.FilterSpans(traceId, spanDetailsMap, resourceAttrMap, spanAttrMap)
	return workloadIds, groupBy
}

// Execute rules on the span and populate the span details.
func (th *TraceHandler) filterSpansOnZkSpanDetails(traceId string, spanId string, spanDetails model.OTelSpanDetails) ([]string, model.GroupByMap) {
	spanDetailsMap := utils.ObjectToInterfaceMap(spanDetails)
	spanAttrMap := *spanDetails.SpanAttributes
	resourceAttrMap := *spanDetails.ResourceAttributes

	logger.Debug(traceLogTag, "Performing span filtering on ZK span ", spanId)
	workloadIds, groupBy := th.spanFilteringHandler.FilterSpans(traceId, spanDetailsMap, resourceAttrMap, spanAttrMap)
	return workloadIds, groupBy
}

// Generate Span details from the span.
func (th *TraceHandler) generateSpanDetails(span *tracev1.Span, schemaVersion string, resourceAttrMap model.GenericMap, scopeAttrMap model.GenericMap) model.OTelSpanDetails {
	spanAttrMap := utils.ConvertKVListToMap(span.Attributes)
	spanDetails := th.createSpanDetails(span, resourceAttrMap, spanAttrMap)
	spanDetails.SchemaVersion = schemaVersion

	/* Populate attributes */
	if th.otlpConfig.SetSpanAttributes {
		spanDetails.SpanAttributes = model.GenericMapPtrFromMap(spanAttrMap)
		spanDetails.ScopeAttributes = model.GenericMapPtrFromMap(scopeAttrMap)
		spanDetails.ResourceAttributes = model.GenericMapPtrFromMap(resourceAttrMap)
	}

	spanDetailsMap := utils.ObjectToInterfaceMap(spanDetails)

	/* Detect Span protocol */
	executorAttrStore := th.factory.GetExecutorAttrStore()
	podDetailsStore := th.factory.GetPodDetailsStore()
	protocolIdentifierStoreKey, _ := cache.CreateKey(ExecutorModel.ExecutorOTel, spanDetails.SchemaVersion, ExecutorModel.ProtocolIdentifier)
	identifierProtocolUtil := utils.NewSpanProtocolUtil(&spanDetails, &spanDetailsMap, executorAttrStore, podDetailsStore, &protocolIdentifierStoreKey)
	spanDetails.Protocol = identifierProtocolUtil.DetectSpanProtocol()

	/* Populate Span protocol attributes */
	executorProtocol := utils.GetExecutorProtocolFromSpanProtocol(spanDetails.Protocol)
	attrStoreKey, _ := cache.CreateKey(ExecutorModel.ExecutorOTel, spanDetails.SchemaVersion, executorProtocol)
	spanProtocolUtil := utils.NewSpanProtocolUtil(&spanDetails, &spanDetailsMap, executorAttrStore, podDetailsStore, &attrStoreKey)
	spanProtocolUtil.AddSpanProtocolProperties()

	return spanDetails
}

// Populate Span common properties.
func (th *TraceHandler) createSpanDetails(span *tracev1.Span, resourceAttrMap map[string]interface{}, spanAttrMap map[string]interface{}) model.OTelSpanDetails {
	spanDetail := model.OTelSpanDetails{}
	//spanDetail.TraceId = hex.EncodeToString(span.TraceId)
	//spanDetail.SpanId = hex.EncodeToString(span.SpanId)
	spanDetail.SetParentSpanId(hex.EncodeToString(span.ParentSpanId))
	spanDetail.SpanKind = utils.GetSpanKind(span.Kind)
	spanDetail.StartNs = span.StartTimeUnixNano
	spanDetail.LatencyNs = span.EndTimeUnixNano - span.StartTimeUnixNano
	spanDetail.SpanName = span.Name

	if serviceName, getSvcNameOk := resourceAttrMap[common.OTelResourceServiceName]; getSvcNameOk {
		if serviceNameStr, typeConvOk := serviceName.(string); typeConvOk {
			spanDetail.ServiceName = serviceNameStr
		}
	}

	if len(span.Events) > 0 {
		for _, event := range span.Events {
			if event.Name == common.OTelSpanEventException {
				spanExceptionDetails := th.createExceptionDetails(span, event)
				spanDetail.Errors = append(spanDetail.Errors, spanExceptionDetails)
			}
		}
	}

	sourceIp, destIp := utils.GetSourceDestIPPair(spanDetail.SpanKind, spanAttrMap, resourceAttrMap)
	podDetailsStore := th.factory.GetPodDetailsStore()
	if len(sourceIp) > 0 {
		spanDetail.SourceIp = &sourceIp
		source := podDetails.GetServiceNameFromPodDetailsStore(sourceIp, podDetailsStore)
		if len(source) > 0 {
			spanDetail.Source = &source
		}
	}
	if len(destIp) > 0 {
		spanDetail.DestIp = &destIp
		dest := podDetails.GetServiceNameFromPodDetailsStore(destIp, podDetailsStore)
		if len(dest) > 0 {
			spanDetail.Destination = &dest
		}
	}

	return spanDetail
}

func (th *TraceHandler) createExceptionDetails(span *tracev1.Span, event *tracev1.Span_Event) model.SpanErrorInfo {
	exceptionDetails := redis.CreateExceptionDetails(event)
	spanIdStr := hex.EncodeToString(span.SpanId)
	hash, err := th.exceptionHandler.SyncExceptionData(exceptionDetails, spanIdStr)
	if err != nil {
		logger.Error(traceLogTag, "Error while syncing exception data for spanId ", spanIdStr, " with error ", err)
	}
	spanExceptionDetails := model.SpanErrorInfo{}
	spanExceptionDetails.ErrorType = model.ErrorTypeException
	spanExceptionDetails.Hash = hash
	spanExceptionDetails.ExceptionType = exceptionDetails.Type
	spanExceptionDetails.Message = exceptionDetails.Message
	return spanExceptionDetails
}

func (th *TraceHandler) deleteFromTraceStore(keysToDelete []string) {
	th.traceStoreMutex.Lock()
	defer th.traceStoreMutex.Unlock()
	for _, key := range keysToDelete {
		th.traceStore.Delete(key)
	}
}

func (th *TraceHandler) addToTraceStore(key string, spanDetails model.OTelSpanDetails) {
	th.traceStoreMutex.Lock()
	defer th.traceStoreMutex.Unlock()
	th.traceStore.Store(key, spanDetails)
}

func (th *TraceHandler) pushSpansToRedisPipeline() []string {
	th.traceStoreMutex.Lock()
	defer th.traceStoreMutex.Unlock()

	var keysToDelete []string

	// Iterate over the sync.Map and push the data to redis
	th.traceStore.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		spanDetails := value.(model.OTelSpanDetails)
		//Split keyStr using delimiter.
		ids := strings.Split(keyStr, delimiter)
		if len(ids) == 2 {
			traceIDStr := ids[0]
			spanIDStr := ids[1]
			err := th.traceRedisHandler.PutTraceData(traceIDStr, spanIDStr, spanDetails)
			if err != nil {
				logger.Debug(traceLogTag, "Error while putting trace data to redis ", err)
				// Returning false to stop the iteration
				return false
			}
		}
		keysToDelete = append(keysToDelete, keyStr)
		return true
	})
	return keysToDelete
}
