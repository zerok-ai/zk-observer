package handler

import (
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-observer/common"
	"github.com/zerok-ai/zk-observer/config"
	"github.com/zerok-ai/zk-observer/model"
	"github.com/zerok-ai/zk-observer/stores/badger"
	"github.com/zerok-ai/zk-observer/stores/redis"
	"github.com/zerok-ai/zk-observer/utils"
	_ "github.com/zerok-ai/zk-utils-go/common"
	zkUtilsCommonModel "github.com/zerok-ai/zk-utils-go/common"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/podDetails"
	zkUtilsEnrichedSpan "github.com/zerok-ai/zk-utils-go/proto/enrichedSpan"
	zkUtilsOtel "github.com/zerok-ai/zk-utils-go/proto/opentelemetry"
	ExecutorModel "github.com/zerok-ai/zk-utils-go/scenario/model"
	"github.com/zerok-ai/zk-utils-go/scenario/model/evaluators/cache"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"io"
	"strings"
	"sync"
)

var traceLogTag = "TraceHandler"
var delimiter = "__"
var DefaultNodeJsSchemaUrl = "https://opentelemetry.io/schemas/1.7.0"

type SpanForStorage interface {
	zkUtilsEnrichedSpan.OtelEnrichedRawSpan | model.OTelSpanDetails
}

type TraceHandler struct {
	traceStore                   sync.Map
	traceStoreMutex              sync.Mutex
	traceRedisHandler            *redis.TraceRedisHandler
	traceBadgerHandler           *badger.TraceBadgerHandler
	exceptionHandler             *redis.ExceptionRedisHandler
	resourceDetailsHandler       *redis.ResourceRedisHandler
	serviceListHandler           *redis.ServiceListRedisHandler
	resourceAndScoperAttrHandler *redis.ResourceAndScopeAttributesHandler
	otlpConfig                   *config.OtlpConfig
	spanFilteringHandler         *redis.SpanFilteringHandler
	factory                      stores.StoreFactory
}

func NewTraceHandler(config *config.OtlpConfig, factory stores.StoreFactory) (*TraceHandler, error) {
	handler := TraceHandler{}
	traceRedisHandler, err := redis.NewTracesRedisHandler(config)
	if err != nil {
		logger.Error(traceLogTag, "Error while creating redis handler:", err)
		return nil, err
	}

	traceBadgerHandler, err := badger.NewTracesBadgerHandler(config)
	if err != nil {
		logger.Error(traceLogTag, "Error while creating badger handler:", err)
		return nil, err
	}

	resourceAndScopeAttrRedisHandler, err := redis.NewResourceAndScopeAttributesHandler(config)
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

	serviceListHandler, err := redis.NewServiceListRedisHandler(config)
	if err != nil {
		logger.Error(traceLogTag, "Error while creating service list handler:", err)
		return nil, err
	}

	executorAttrStore := factory.GetExecutorAttrStore()
	podDetailsStore := factory.GetPodDetailsStore()

	handler.resourceDetailsHandler = resourceHandler
	handler.exceptionHandler = exceptionHandler
	handler.traceStore = sync.Map{}
	handler.traceRedisHandler = traceRedisHandler
	handler.traceBadgerHandler = traceBadgerHandler
	handler.otlpConfig = config
	handler.factory = factory
	handler.resourceAndScoperAttrHandler = resourceAndScopeAttrRedisHandler
	handler.serviceListHandler = serviceListHandler
	spanFilteringHandler, err := redis.NewSpanFilteringHandler(config, executorAttrStore, podDetailsStore)
	if err != nil {
		logger.Error(traceLogTag, "Error while creating span filtering handler:", err)
		return nil, err
	}
	handler.spanFilteringHandler = spanFilteringHandler

	return &handler, nil
}

func (th *TraceHandler) ServeHTTP(ctx iris.Context) {
	// Read the request body
	body, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		return
	}

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
	th.traceBadgerHandler.SyncPipeline()
	th.exceptionHandler.SyncPipeline()
	th.resourceDetailsHandler.SyncPipeline()
	th.spanFilteringHandler.SyncPipeline()
	th.resourceAndScoperAttrHandler.SyncPipeline()
	th.serviceListHandler.SyncPipeline()
}

func (th *TraceHandler) processOTelSpanEvents(span *tracev1.Span) ([]zkUtilsCommonModel.GenericMap, bool) {
	var spanEventsList []zkUtilsCommonModel.GenericMap
	var errorFlag bool
	if len(span.Events) > 0 {
		for _, event := range span.Events {
			eventMap := utils.ObjectToInterfaceMap(event)
			if event.Name == common.OTelSpanEventException {
				hash := th.processOTelSpanException(hex.EncodeToString(span.SpanId), event)
				// override attributes with nil as data is saved to other db
				eventMap[common.OTelSpanEventAttrKey] = nil
				eventMap[common.OTelSpanEventExceptionHashKey] = hash
				errorFlag = true
				spanEventsList = append(spanEventsList, eventMap)
			} else {
				eventAttributes := utils.ConvertKVListToMap(event.Attributes)
				// override attributes with event attributes hashmap
				eventMap[common.OTelSpanEventAttrKey] = eventAttributes
				spanEventsList = append(spanEventsList, eventMap)
			}
		}
	}
	return spanEventsList, errorFlag
}

func (th *TraceHandler) processOTelSpanException(spanIdStr string, event *tracev1.Span_Event) string {
	exceptionDetails := redis.CreateExceptionDetails(event)
	hash, err := th.exceptionHandler.SyncExceptionData(exceptionDetails, spanIdStr)
	if err != nil {
		logger.Error(traceLogTag, "Error while syncing exception data for spanId ", spanIdStr, " with error ", err)
	}
	return hash
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
		resourceInfo := model.ResourceInfo{
			SchemaUrl: schemaUrl,
		}

		resourceInfo.AttributesMap = map[string]interface{}{}
		var resourceAttrHash string
		var resourceAttrMap map[string]interface{}
		var resourceInfoMap map[string]interface{}
		serviceName := common.ScenarioWorkloadGenericServiceNameKey
		if resourceSpan.Resource != nil {
			resourceInfoMap = utils.ObjectToInterfaceMap(resourceInfo)
			resourceAttrMap = utils.ConvertKVListToMap(resourceSpan.Resource.Attributes)
			resourceInfoMap["attributes_map"] = resourceAttrMap
			resourceAttrHash = utils.ResourceAttributeHashPrefix + utils.GetMD5OfMap(resourceInfoMap)
			serviceName = utils.GetServiceName(resourceAttrMap)
		}
		for _, scopeSpans := range resourceSpan.ScopeSpans {
			scopeInfo := model.ScopeInfo{
				Name:      scopeSpans.Scope.Name,
				Version:   scopeSpans.Scope.Version,
				SchemaUrl: scopeSpans.SchemaUrl,
			}
			scopeInfo.AttributesMap = map[string]interface{}{}

			var scopeAttrHash string
			var scopeInfoMap map[string]interface{}
			var scopeAttrMap map[string]interface{}
			if scopeSpans.Scope != nil {
				scopeInfoMap = utils.ObjectToInterfaceMap(scopeInfo)
				scopeAttrMap = utils.ConvertKVListToMap(scopeSpans.Scope.Attributes)
				scopeInfoMap["attributes_map"] = scopeAttrMap
				scopeAttrHash = utils.ScopeAttributeHashPrefix + utils.GetMD5OfMap(scopeInfoMap)
			}

			for _, span := range scopeSpans.Spans {
				//TODO: remove this later
				span.Links = nil

				processedSpanCount++
				traceId := hex.EncodeToString(span.TraceId)
				spanId := hex.EncodeToString(span.SpanId)
				if traceId == "" || spanId == "" {
					logger.Warn(traceLogTag, "TraceId or SpanId is empty for span ", spanId)
					continue
				}

				key := traceId + delimiter + spanId
				var resourceIp string
				spanAttributes := utils.ConvertKVListToMap(span.Attributes)
				spanJSON := utils.ObjectToInterfaceMap(span)
				spanJSON[common.OTelLatencyNsKey] = span.EndTimeUnixNano - span.StartTimeUnixNano
				spanJSON[common.OTelSpanAttrKey] = spanAttributes
				spanJSON[common.OTelResourceAttrKey] = resourceAttrMap
				spanJSON[common.OTelScopeAttrKey] = scopeAttrMap
				spanJSON[common.OTelSchemaVersionKey] = schemaVersion
				spanEvents, errorFlag := th.processOTelSpanEvents(span)
				spanJSON[common.OTelSpanEventsKey] = spanEvents
				spanJSON[common.OTelSpanErrorKey] = errorFlag
				// Evaluating and storing data in Otel span format.
				workloadIds, groupBy := th.spanFilteringHandler.FilterSpans(traceId, spanJSON, serviceName)

				spanKind := model.NewFromOTelSpan(span.Kind)
				sourceIP, destIP := utils.GetSourceDestIPPair(spanKind, spanAttributes, resourceAttrMap)
				resourceIp = utils.GetResourceIp(spanKind, sourceIP, destIP)

				span.Attributes = nil
				span.Events = nil
				enrichedRawSpan := zkUtilsEnrichedSpan.OtelEnrichedRawSpan{
					Span:                   span,
					SpanEvents:             spanEvents,
					SpanAttributes:         spanAttributes,
					ResourceAttributesHash: resourceAttrHash,
					ScopeAttributesHash:    scopeAttrHash,
					WorkloadIdList:         workloadIds,
					GroupBy:                groupBy,
				}

				spanDetailsProtobufType := enrichedRawSpan.GetProtoEnrichedSpan()
				th.addEnrichedSpanToTraceStore(key, spanDetailsProtobufType)
				if err := th.resourceDetailsHandler.SyncResourceData(resourceIp, resourceAttrMap); err != nil {
					logger.Error(traceLogTag, "Error while saving resource data to redis for spanId ", spanId, " error: ", err)
				}

				if err := th.resourceAndScoperAttrHandler.SyncResourceAndScopeAttrData(resourceAttrHash, resourceInfoMap); err != nil {
					logger.Error(traceLogTag, "Error while saving resource  data to redis for spanId ", spanId, " error: ", err)
				}

				if err := th.resourceAndScoperAttrHandler.SyncResourceAndScopeAttrData(scopeAttrHash, scopeInfoMap); err != nil {
					logger.Error(traceLogTag, "Error while saving  scope data to redis for spanId ", spanId, " error: ", err)
				}

				logger.Debug(traceLogTag, "service name:", serviceName)
				if serviceName == common.ScenarioWorkloadGenericServiceNameKey {
					logger.ErrorF(traceLogTag, "Service name could not be fetched for spanId %s, traceId %s", spanId, traceId)
				} else if err := th.serviceListHandler.PutServiceListData(common.ServiceListKey, serviceName); err != nil {
					logger.Error(traceLogTag, "Error while saving service list data to redis for spanId ", spanId, " error: ", err)
				}
			}
		}
	}
	defer logger.InfoF(traceLogTag, "Processed %v spans", processedSpanCount)
}

func (th *TraceHandler) EvalAndStoreZkSpan() {

}

// Generate Span details from the span.
func (th *TraceHandler) generateSpanDetails(span *tracev1.Span, schemaVersion string, resourceAttrMap zkUtilsCommonModel.GenericMap, resourceAttrHash string, scopeAttrMap zkUtilsCommonModel.GenericMap, scopeAttrHash string) model.OTelSpanDetails {
	spanAttrMap := utils.ConvertKVListToMap(span.Attributes)
	spanDetails := th.createSpanDetails(span, resourceAttrMap, spanAttrMap)
	spanDetails.SchemaVersion = schemaVersion

	/* Populate attributes */
	if th.otlpConfig.SetSpanAttributes {
		spanDetails.SpanAttributes = model.GenericMapPtrFromMap(spanAttrMap)
		spanDetails.ResourceAttributes = &resourceAttrMap
		spanDetails.ScopeAttributes = &scopeAttrMap

		spanDetails.ScopeAttributesHash = scopeAttrHash
		spanDetails.ResourceAttributesHash = resourceAttrHash
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

func (th *TraceHandler) addEnrichedSpanToTraceStore(key string, spanDetails *zkUtilsOtel.OtelEnrichedRawSpanForProto) {
	th.traceStoreMutex.Lock()
	defer th.traceStoreMutex.Unlock()
	spanProto, err := proto.Marshal(spanDetails)
	if err != nil {
		logger.ErrorF(traceLogTag, "Error encoding SpanDetails for spanID %s: %v\n", spanDetails.Span.SpanId, err)
		return
	}

	th.traceStore.Store(key, spanProto)
}

func (th *TraceHandler) addZkSpanToTraceStore(key string, spanDetails model.OTelSpanDetails) {
	th.traceStoreMutex.Lock()
	defer th.traceStoreMutex.Unlock()
	// Clear scope and Resource attributes from TraceStore as they are not required in zk span.
	spanDetails.ResourceAttributes = nil
	spanDetails.ScopeAttributes = nil
	th.traceStore.Store(key, spanDetails)
}

func (th *TraceHandler) pushSpansToRedisPipeline() []string {
	th.traceStoreMutex.Lock()
	defer th.traceStoreMutex.Unlock()

	var keysToDelete []string

	// Iterate over the sync.Map and push the data to redis
	th.traceStore.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		//Split keyStr using delimiter.
		ids := strings.Split(keyStr, delimiter)
		if len(ids) != 2 {
			logger.Error(traceLogTag, "Error while splitting key ", keyStr)
			return true
		}
		traceIDStr := ids[0]
		spanIDStr := ids[1]

		err := th.traceBadgerHandler.PutTraceData(traceIDStr, spanIDStr, value.([]byte))
		if err != nil {
			logger.Debug(traceLogTag, "Error while putting trace data to badger ", err)
			// Returning false to stop the iteration
			return false
		}

		err = th.traceRedisHandler.PutTraceSource(traceIDStr, spanIDStr)
		if err != nil {
			logger.Debug(traceLogTag, "Error while putting trace source to redis ", err)
			// Returning false to stop the iteration
			return false
		}

		keysToDelete = append(keysToDelete, keyStr)
		return true
	})
	return keysToDelete
}

func (th *TraceHandler) GetBulkDataFromBadgerForPrefix(prefixList []string) (*zkUtilsOtel.BadgerResponseList, error) {
	traceToDataMap, err := th.traceBadgerHandler.GetBulkDataForPrefixList(prefixList)
	var resp *zkUtilsOtel.BadgerResponseList
	if err != nil {
		logger.Error(traceLogTag, "Error while getting data from badger for prefix list ", prefixList, " error is ", err)
		return resp, err
	}

	if len(traceToDataMap) > 0 {
		resp = zkUtilsCommonModel.ToPtr(zkUtilsOtel.BadgerResponseList{ResponseList: make([]*zkUtilsOtel.BadgerResponse, 0)})
	}

	for k, v := range traceToDataMap {
		var d zkUtilsOtel.BadgerResponse
		d.Key = k
		d.Value = v
		resp.ResponseList = append(resp.ResponseList, &d)
	}

	return resp, nil

}
