package handler

import (
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/redis"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"io"
	"strings"
	"sync"
)

var traceLogTag = "TraceHandler"
var delimiter = "__"
var DefaultNodeJsSchemaUrl = "https://opentelemetry.io/schemas/1.7.0"

type TraceHandler struct {
	traceStore             sync.Map
	traceRedisHandler      *redis.TraceRedisHandler
	exceptionHandler       *redis.ExceptionRedisHandler
	resourceDetailsHandler *redis.ResourceRedisHandler
	otlpConfig             *config.OtlpConfig
	spanFilteringHandler   *redis.SpanFilteringHandler
}

func NewTraceHandler(config *config.OtlpConfig) (*TraceHandler, error) {
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

	handler.resourceDetailsHandler = resourceHandler
	handler.exceptionHandler = exceptionHandler
	handler.traceStore = sync.Map{}
	handler.traceRedisHandler = traceRedisHandler
	handler.otlpConfig = config
	spanFilteringHandler, err := redis.NewSpanFilteringHandler(config)
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
	err = th.PushDataToRedis()
	if err != nil {
		logger.Error(traceLogTag, "Error while pushing data to redis ", err)
		return
	}

	// Respond to the client
	ctx.StatusCode(iris.StatusOK)
}

func (th *TraceHandler) PushDataToRedis() error {
	var keysToDelete []string
	var err error

	th.traceStore.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		spanDetails := value.(map[string]interface{})
		//Split keyStr using delimiter.
		ids := strings.Split(keyStr, delimiter)
		if len(ids) == 2 {
			traceIDStr := ids[0]
			spanIDStr := ids[1]
			err = th.traceRedisHandler.PutTraceData(traceIDStr, spanIDStr, spanDetails)
			if err != nil {
				logger.Debug(traceLogTag, "Error whole putting trace data to redis ", err)
				// Returning false to stop the iteration
				return false
			}
		}
		keysToDelete = append(keysToDelete, keyStr)
		return true
	})

	// Delete the keys from the sync.Map after the iteration
	for _, key := range keysToDelete {
		th.traceStore.Delete(key)
	}

	th.traceRedisHandler.SyncPipeline()
	th.exceptionHandler.SyncPipeline()
	th.resourceDetailsHandler.SyncPipeline()
	th.spanFilteringHandler.SyncPipeline()

	return err
}

func (th *TraceHandler) ProcessTraceData(resourceSpans []*tracev1.ResourceSpans) []map[string]interface{} {
	var spanDetails []map[string]interface{}
	if len(resourceSpans) == 0 {
		return spanDetails
	}
	for _, resourceSpan := range resourceSpans {
		schemaUrl := resourceSpan.SchemaUrl
		resourceAttrMap := map[string]interface{}{}
		if resourceSpan.Resource != nil {
			resourceAttrMap = utils.ConvertKVListToMap(resourceSpan.Resource.Attributes)
		}
		for _, scopeSpans := range resourceSpan.ScopeSpans {
			for _, span := range scopeSpans.Spans {
				traceId := hex.EncodeToString(span.TraceId)
				spanId := hex.EncodeToString(span.SpanId)
				logger.Debug(traceLogTag, "traceId", traceId, " , spanId", spanId, " , spanKind ", span.Kind, " ,parentSpanId ", hex.EncodeToString(span.ParentSpanId))

				spanDetails := th.createSpanDetails(span, resourceAttrMap)
				if len(schemaUrl) == 0 {
					language, ok := resourceAttrMap[common.ResourceLanguageKey]
					if ok {
						languageStr := language.(string)
						if languageStr == "nodejs" {
							schemaUrl = DefaultNodeJsSchemaUrl
						}
					}
				}

				spanDetails["schema_version"] = utils.GetSchemaVersion(schemaUrl)

				logger.Debug(traceLogTag, "Performing span filtering on span ", spanId)
				//TODO: Make this Async.
				workloadIds := th.spanFilteringHandler.FilterSpans(spanDetails, traceId)
				spanDetails[common.SatisfiedWorkloadIdsKey] = workloadIds

				//Updating the spanDetails in traceStore.
				key := traceId + delimiter + spanId
				th.traceStore.Store(key, spanDetails)

				err := th.resourceDetailsHandler.SyncResourceData(&spanDetails, resourceAttrMap)
				if err != nil {
					logger.Error(traceLogTag, "Error while saving resource data to redis for spanId ", spanId, " error is ", err)
				}
			}
		}
	}
	return spanDetails
}

func (th *TraceHandler) createSpanDetails(span *tracev1.Span, resourceAttrMap map[string]interface{}) map[string]interface{} {
	spanDetail := map[string]interface{}{}
	parentSpanId := hex.EncodeToString(span.ParentSpanId)
	if len(parentSpanId) == 0 {
		parentSpanId = "0000000000000000"
	}
	spanDetail[common.ParentSpanIdKey] = parentSpanId
	spanKind := utils.GetSpanKind(span.Kind)
	spanDetail[common.SpanKindKey] = spanKind

	attrMap := utils.ConvertKVListToMap(span.Attributes)

	if len(span.Events) > 0 {
		for _, event := range span.Events {
			if event.Name == "exception" {
				spanExceptionDetails := th.createExceptionDetails(span, event)
				existingErrorDetails, ok := spanDetail["errors"].([]map[string]interface{})
				if !ok {
					existingErrorDetails = []map[string]interface{}{}
				}
				existingErrorDetails = append(existingErrorDetails, spanExceptionDetails)
				spanDetail["errors"] = existingErrorDetails
			}
		}
	}

	if th.otlpConfig.SetSpanAttributes {
		spanDetail[common.AttributesKey] = attrMap
	}

	sourceIp, destIp := utils.GetSourceDestIPPair(spanKind, attrMap, resourceAttrMap)

	if len(sourceIp) > 0 {
		spanDetail[common.SourceIpKey] = sourceIp
	}
	if len(destIp) > 0 {
		spanDetail[common.DestIpKey] = destIp
	}

	spanDetail[common.StartNsKey] = span.StartTimeUnixNano
	spanDetail[common.LatencyNsKey] = span.EndTimeUnixNano - span.StartTimeUnixNano

	return spanDetail
}

func (th *TraceHandler) createExceptionDetails(span *tracev1.Span, event *tracev1.Span_Event) map[string]interface{} {
	exceptionDetails := redis.CreateExceptionDetails(event)
	spanIdStr := hex.EncodeToString(span.SpanId)
	hash, err := th.exceptionHandler.SyncExceptionData(exceptionDetails, spanIdStr)
	if err != nil {
		logger.Error(traceLogTag, "Error while syncing exception data for spanId ", spanIdStr, " with error ", err)
	}
	spanExceptionDetails := map[string]interface{}{}
	spanExceptionDetails["error_type"] = "exception"
	spanExceptionDetails["hash"] = hash
	spanExceptionDetails["exception_type"] = exceptionDetails.Type
	spanExceptionDetails["message"] = exceptionDetails.Message
	return spanExceptionDetails
}
