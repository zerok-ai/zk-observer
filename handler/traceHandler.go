package handler

import (
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"io"
	"sync"
)

var TRACE_LOG_TAG = "TraceHandler"

var DBSystemAttributeKey = "db.system"
var HTTPMethodAttributeKey = "http.method"
var NetProtocolNameAttributeKey = "net.protocol.name"
var HTTPRouteAttributeKey = "http.route"

type TraceHandler struct {
	traceStore   sync.Map
	redisHandler *utils.RedisHandler
}

func NewTraceHandler(redisHandler *utils.RedisHandler) *TraceHandler {
	handler := TraceHandler{}
	handler.traceStore = sync.Map{}
	handler.redisHandler = redisHandler
	return &handler
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

	th.processTraceData(&traceData)
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

		err = th.redisHandler.PutTraceData(traceIDStr, traceDetails)
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

func (th *TraceHandler) processTraceData(traceData *tracev1.TracesData) []*model.SpanDetails {
	var spanDetails []*model.SpanDetails

	for _, resourceSpans := range traceData.ResourceSpans {
		for _, scopeSpans := range resourceSpans.ScopeSpans {
			for _, span := range scopeSpans.Spans {
				traceId := hex.EncodeToString(span.TraceId)
				spanId := hex.EncodeToString(span.SpanId)
				logger.Debug(TRACE_LOG_TAG, "spanKind", span.Kind)
				logger.Debug(TRACE_LOG_TAG, "traceId", traceId, " , spanId", spanId, " ,parentSpanId ", hex.EncodeToString(span.ParentSpanId))

				spanDetails := th.createSpanDetails(span)

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

func (th *TraceHandler) createSpanDetails(span *tracev1.Span) model.SpanDetails {
	spanDetail := model.SpanDetails{}
	spanDetail.ParentSpanID = hex.EncodeToString(span.ParentSpanId)
	spanDetail.SpanKind = th.getSpanKind(span.Kind)
	attr := span.Attributes
	attrMap := map[string]interface{}{}
	for _, kv := range attr {
		attrMap[kv.Key] = kv.Value
	}
	dbSystemAttrValue, ok := attrMap[DBSystemAttributeKey]
	if ok {
		tmp := dbSystemAttrValue.(*commonv1.AnyValue)
		tmpValue := tmp.Value.(*commonv1.AnyValue_StringValue)
		spanDetail.Protocol = tmpValue.StringValue
	} else {
		netProtocolAttrValue, ok := attrMap[NetProtocolNameAttributeKey]
		if ok {
			tmp := netProtocolAttrValue.(*commonv1.AnyValue)
			tmpValue := tmp.Value.(*commonv1.AnyValue_StringValue)
			spanDetail.Protocol = tmpValue.StringValue
		}
	}

	//logger.Debug(TRACE_LOG_TAG, "Attr Map ", attrMap)
	return spanDetail
}

func (th *TraceHandler) getSpanKind(kind tracev1.Span_SpanKind) model.SpanKind {
	switch kind {
	case tracev1.Span_SPAN_KIND_INTERNAL:
		return model.SpanKindInternal
	case tracev1.Span_SPAN_KIND_SERVER:
		return model.SpanKindServer
	case tracev1.Span_SPAN_KIND_PRODUCER:
		return model.SpanKindProducer
	case tracev1.Span_SPAN_KIND_CONSUMER:
		return model.SpanKindConsumer
	case tracev1.Span_SPAN_KIND_CLIENT:
		return model.SpanKindClient
	}
	return model.SpanKindInternal
}
