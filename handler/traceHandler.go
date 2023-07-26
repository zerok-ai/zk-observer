package handler

import (
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	v1 "github.com/zerok-ai/zk-otlp-receiver/proto/go.opentelemetry.io/proto/otlp/trace/v1"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"io"
	"net/http"
)

var TRACE_LOG_TAG = "TraceHandler"

var DBSystemAttributeKey = "db.system"
var HTTPMethodAttributeKey = "http.method"
var NetProtocolNameAttributeKey = "net.protocol.name"
var HTTPRouteAttributeKey = "http.route"

type TraceHandler struct {
	traceStore   map[string]model.TraceDetails
	redisHandler *utils.RedisHandler
}

func NewTraceHandler(redisHandler *utils.RedisHandler) *TraceHandler {
	handler := TraceHandler{}
	handler.traceStore = map[string]model.TraceDetails{}
	handler.redisHandler = redisHandler
	return &handler
}

func (th *TraceHandler) HandleTraceRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	//logger.Debug(TRACE_LOG_TAG, string(body))

	// Unmarshal the protobuf message from the request body
	var traceData v1.TracesData
	err = proto.Unmarshal(body, &traceData)
	if err != nil {
		http.Error(w, "Failed to unmarshal proto data", http.StatusBadRequest)
		return
	}

	th.processTraceData(&traceData)

	// Respond to the client
	w.WriteHeader(http.StatusOK)
	_, err = fmt.Fprintln(w, "Trace data received successfully!")
	if err != nil {
		return
	}
}
func (th *TraceHandler) pushDataToRedis() error {
	var keysToDelete []string
	var err error
	for traceID, traceDetails := range th.traceStore {
		err = th.redisHandler.PutTraceData(traceID, traceDetails)
		if err != nil {
			logger.Debug(TRACE_LOG_TAG, "Error whole putting trace data to redis ", err)
			break
		}
		keysToDelete = append(keysToDelete, traceID)
	}
	for _, key := range keysToDelete {
		delete(th.traceStore, key)
	}
	return err
}

func (th *TraceHandler) processTraceData(traceData *v1.TracesData) []*model.SpanDetails {
	var spanDetails []*model.SpanDetails

	for _, resourceSpans := range traceData.ResourceSpans {
		for _, scopeSpans := range resourceSpans.ScopeSpans {
			for _, span := range scopeSpans.Spans {
				traceId := hex.EncodeToString(span.TraceId)
				spanId := hex.EncodeToString(span.SpanId)
				logger.Debug(TRACE_LOG_TAG, "spanKind", span.Kind)
				logger.Debug(TRACE_LOG_TAG, "traceId", traceId, " , spanId", spanId, " ,parentSpanId ", hex.EncodeToString(span.ParentSpanId))

				spanDetails := th.createSpanDetails(span)
				traceDetails, found := th.traceStore[traceId]

				if !found {
					traceDetails = model.TraceDetails{
						SpanDetailsMap: make(map[string]model.SpanDetails),
					}
				}
				traceDetails.SpanDetailsMap[spanId] = spanDetails
				th.traceStore[traceId] = traceDetails
			}
		}
	}
	return spanDetails
}

func (th *TraceHandler) createSpanDetails(span *v1.Span) model.SpanDetails {
	spanDetail := model.SpanDetails{}
	spanDetail.ParentSpanID = string(span.ParentSpanId)
	spanDetail.Endpoint = ""
	spanDetail.LocalEndpoint = model.Endpoint{}
	spanDetail.SpanKind = th.getSpanKind(span.Kind)
	//attr := span.Attributes
	//attrMap := map[string]interface{}{}
	//for _, kv := range attr {
	//	attrMap[kv.Key] = kv.Value
	//}
	//dbSystemAttrValue, ok := attrMap[DBSystemAttributeKey]
	//if ok {
	//	spanDetail.Protocol = dbSystemAttrValue
	//} else {
	//
	//}

	//logger.Debug(TRACE_LOG_TAG, "Attr Map ", attrMap)
	return spanDetail
}

func (th *TraceHandler) getSpanKind(kind v1.Span_SpanKind) model.SpanKind {
	switch kind {
	case v1.Span_SPAN_KIND_INTERNAL:
		return model.SpanKindInternal
	case v1.Span_SPAN_KIND_SERVER:
		return model.SpanKindServer
	case v1.Span_SPAN_KIND_PRODUCER:
		return model.SpanKindProducer
	case v1.Span_SPAN_KIND_CONSUMER:
		return model.SpanKindConsumer
	case v1.Span_SPAN_KIND_CLIENT:
		return model.SpanKindClient
	}
	return model.SpanKindClient
}
