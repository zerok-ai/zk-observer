package handler

import (
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	v1 "github.com/zerok-ai/zk-otlp-receiver/proto/go.opentelemetry.io/proto/otlp/trace/v1"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"io"
	"net/http"
)

var TRACE_LOG_TAG = "TraceHandler"

var DBSystemAttributeKey = "db.system"
var HTTPMethodAttributeKey = "http.method"
var NetProtocolNameAttributeKey = "net.protocol.name"
var HTTPRouteAttributeKey = "http.route"

func HandleTraceRequest(w http.ResponseWriter, r *http.Request) {
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

	getSpanDetails(&traceData)

	// Respond to the client
	w.WriteHeader(http.StatusOK)
	_, err = fmt.Fprintln(w, "Trace data received successfully!")
	if err != nil {
		return
	}
}

func getSpanDetails(traceData *v1.TracesData) []*model.SpanDetails {
	var spanDetails []*model.SpanDetails

	for _, resourceSpans := range traceData.ResourceSpans {
		for _, scopeSpans := range resourceSpans.ScopeSpans {
			for _, span := range scopeSpans.Spans {
				logger.Debug(TRACE_LOG_TAG, "spanKind", span.Kind)
				logger.Debug(TRACE_LOG_TAG, "traceId", hex.EncodeToString(span.TraceId), " , spanId", hex.EncodeToString(span.SpanId), " ,parentSpanId ", hex.EncodeToString(span.ParentSpanId))
				createSpanDetails(span)
			}
		}
	}
	return spanDetails
}

func createSpanDetails(span *v1.Span) model.SpanDetails {
	spanDetail := model.SpanDetails{}
	spanDetail.ParentSpanID = string(span.ParentSpanId)
	spanDetail.Endpoint = ""
	spanDetail.LocalEndpoint = model.Endpoint{}
	spanDetail.SpanKind = getSpanKind(span.Kind)
	attr := span.Attributes
	attrMap := map[string]interface{}{}
	for _, kv := range attr {
		attrMap[kv.Key] = kv.Value
	}
	//dbSystemAttrValue, ok := attrMap[DBSystemAttributeKey]
	//if ok {
	//	spanDetail.Protocol = dbSystemAttrValue
	//} else {
	//
	//}

	logger.Debug(TRACE_LOG_TAG, "Attr Map ", attrMap)
	return spanDetail
}

func getSpanKind(kind v1.Span_SpanKind) model.SpanKind {
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
