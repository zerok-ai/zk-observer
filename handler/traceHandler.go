package handler

import (
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"io"
	"sync"
)

var TRACE_LOG_TAG = "TraceHandler"

type TraceHandler struct {
	traceStore        sync.Map
	traceRedisHandler *utils.TraceRedisHandler
	exceptionHandler  *ExceptionHandler
	otlpConfig        *config.OtlpConfig
}

func NewTraceHandler(config *config.OtlpConfig) (*TraceHandler, error) {
	handler := TraceHandler{}
	traceRedisHandler, err := utils.NewTracesRedisHandler(&config.Redis)
	if err != nil {
		logger.Error(TRACE_LOG_TAG, "Error while creating redis handler:", err)
		return nil, err
	}

	exceptionHandler, err := NewExceptionHandler(config)
	if err != nil {
		logger.Error(TRACE_LOG_TAG, "Error while creating exception handler:", err)
		return nil, err
	}

	handler.exceptionHandler = exceptionHandler
	handler.traceStore = sync.Map{}
	handler.traceRedisHandler = traceRedisHandler
	handler.otlpConfig = config
	return &handler, nil
}

func (th *TraceHandler) ServeHTTP(ctx iris.Context) {

	// Read the request body
	body, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		return
	}

	//logger.Debug(TRACE_LOG_TAG, string(body))

	// Unmarshal the protobuf message from the request body
	var traceData tracev1.TracesData
	err = proto.Unmarshal(body, &traceData)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		return
	}

	th.processTraceData(&traceData, ctx)
	err = th.pushDataToRedis()
	if err != nil {
		logger.Error(TRACE_LOG_TAG, "Error while pushing data to redis ", err)
		return
	}

	// Respond to the client
	ctx.StatusCode(iris.StatusOK)
}

func (th *TraceHandler) pushDataToRedis() error {
	var keysToDelete []string
	var err error

	th.traceStore.Range(func(traceID, value interface{}) bool {
		traceIDStr := traceID.(string)
		traceDetails := value.(*model.TraceDetails)

		err = th.traceRedisHandler.PutTraceData(traceIDStr, traceDetails)
		if err != nil {
			logger.Debug(TRACE_LOG_TAG, "Error whole putting trace data to redis ", err)
			// Returning false to stop the iteration
			return false
		}

		keysToDelete = append(keysToDelete, traceIDStr)
		return true
	})

	// Delete the keys from the sync.Map after the iteration
	for _, key := range keysToDelete {
		th.traceStore.Delete(key)
	}

	return err
}

func (th *TraceHandler) processTraceData(traceData *tracev1.TracesData, ctx iris.Context) []*model.SpanDetails {
	var spanDetails []*model.SpanDetails

	for _, resourceSpans := range traceData.ResourceSpans {
		attrMap := map[string]interface{}{}
		if resourceSpans.Resource != nil {
			attrMap = utils.ConvertKVListToMap(resourceSpans.Resource.Attributes)
		}
		for _, scopeSpans := range resourceSpans.ScopeSpans {
			for _, span := range scopeSpans.Spans {
				traceId := hex.EncodeToString(span.TraceId)
				spanId := hex.EncodeToString(span.SpanId)
				logger.Debug(TRACE_LOG_TAG, "spanKind", span.Kind)
				logger.Debug(TRACE_LOG_TAG, "traceId", traceId, " , spanId", spanId, " ,parentSpanId ", hex.EncodeToString(span.ParentSpanId))

				spanDetails := th.createSpanDetails(span, ctx)

				spanDetails.ResourceAttr = attrMap

				traceDetailsPresent, _ := th.traceStore.LoadOrStore(traceId, &model.TraceDetails{
					SpanDetailsMap: sync.Map{},
				})

				traceDetails := traceDetailsPresent.(*model.TraceDetails)

				//Adding the new span details to traceDetails
				traceDetails.SetSpanDetails(spanId, spanDetails)

				//Updating the traceDetails in traceStore.
				th.traceStore.Store(traceId, traceDetails)
			}
		}
	}
	return spanDetails
}

func (th *TraceHandler) createSpanDetails(span *tracev1.Span, ctx iris.Context) model.SpanDetails {
	spanDetail := model.SpanDetails{}
	spanDetail.ParentSpanID = hex.EncodeToString(span.ParentSpanId)
	if len(spanDetail.ParentSpanID) == 0 {
		spanDetail.ParentSpanID = "0000000000000000"
	}
	spanDetail.SpanKind = utils.GetSpanKind(span.Kind)

	attrMap := utils.ConvertKVListToMap(span.Attributes)

	logger.Debug(TRACE_LOG_TAG, "Attribute values ", attrMap)
	if len(span.Events) > 0 {
		for _, event := range span.Events {
			if event.Name == "exception" {
				exceptionDetails := CreateExceptionDetails(event)
				spanIdStr := hex.EncodeToString(span.SpanId)
				hash, err := th.exceptionHandler.SyncExceptionData(exceptionDetails, spanIdStr)
				if err != nil {
					logger.Error(TRACE_LOG_TAG, "Error while syncing exception data for spanId ", spanIdStr, " with error ", err)
				}
				spanExceptionDetails := model.SpanDetailsException{
					Hash:    hash,
					Type:    exceptionDetails.Type,
					Message: exceptionDetails.Message,
				}
				spanDetail.Exception = spanExceptionDetails
			}
		}
	}

	if th.otlpConfig.SetSpanAttributes {
		spanDetail.Attributes = attrMap
	}

	sourceIp, destIp := utils.GetSourceDestIPPair(spanDetail.SpanKind, attrMap, ctx)

	if len(sourceIp) > 0 {
		spanDetail.SourceIP = sourceIp
	}
	if len(destIp) > 0 {
		spanDetail.DestIP = destIp
	}

	spanDetail.StartNs = span.StartTimeUnixNano
	spanDetail.EndNs = span.EndTimeUnixNano

	return spanDetail
}
