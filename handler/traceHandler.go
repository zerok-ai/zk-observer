package handler

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	zkcommon "github.com/zerok-ai/zk-utils-go/common"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"io"
	"net"
	"net/http"
	"sync"
)

var TRACE_LOG_TAG = "TraceHandler"
var NET_SOCK_HOST_ADDR = "net.sock.host.addr"
var NET_SOCK_PEER_ADDR = "net.sock.peer.addr"
var NET_PEER_NAME = "net.peer.name"

type TraceHandler struct {
	traceStore        sync.Map
	traceRedisHandler *utils.TraceRedisHandler
	//TODO: Should we also write a ticker to sync the data from redis?
	existingExceptionData sync.Map
	exceptionRedisHandler *utils.RedisHandler
	otlpConfig            *config.OtlpConfig
}

func NewTraceHandler(config *config.OtlpConfig) (*TraceHandler, error) {
	handler := TraceHandler{}
	traceRedisHandler, err := utils.NewTracesRedisHandler(&config.Redis)
	if err != nil {
		logger.Error(TRACE_LOG_TAG, "Error while creating redis handler:", err)
		return nil, err
	}

	exceptionRedisHandler, err := utils.NewRedisHandler(&config.Redis, "exception")
	if err != nil {
		logger.Error(TRACE_LOG_TAG, "Error while creating exception redis handler:", err)
		return nil, err
	}

	handler.exceptionRedisHandler = exceptionRedisHandler
	handler.traceStore = sync.Map{}
	handler.existingExceptionData = sync.Map{}
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
			attrMap = th.convertKVListToMap(resourceSpans.Resource.Attributes)
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

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return host
}

func (th *TraceHandler) getSourceDestIPPair(spanKind model.SpanKind, attributes map[string]interface{}, ctx iris.Context) (string, string) {
	destIP := ""
	sourceIP := ""

	if spanKind == model.SpanKindClient {
		if len(attributes) > 0 {
			sourceIP = getClientIP(ctx.Request())
			logger.Debug(TRACE_LOG_TAG, "Source Ip for client span  is ", sourceIP)
			if peerAddr, ok := attributes[NET_SOCK_PEER_ADDR]; ok {
				destIP = peerAddr.(string)
			} else if peerName, ok := attributes[NET_PEER_NAME]; ok {
				address, err := net.LookupHost(peerName.(string))
				if err == nil && len(address) > 0 {
					destIP = address[0]
				}
			}
		}
	} else if spanKind == model.SpanKindServer {
		if len(attributes) > 0 {
			if hostAddr, ok := attributes[NET_SOCK_HOST_ADDR]; ok {
				destIP = hostAddr.(string)
			}
			if peerAddr, ok := attributes[NET_SOCK_PEER_ADDR]; ok {
				sourceIP = peerAddr.(string)
			}
		}
	}

	return sourceIP, destIP
}

func (th *TraceHandler) createSpanDetails(span *tracev1.Span, ctx iris.Context) model.SpanDetails {
	spanDetail := model.SpanDetails{}
	spanDetail.ParentSpanID = hex.EncodeToString(span.ParentSpanId)
	if len(spanDetail.ParentSpanID) == 0 {
		spanDetail.ParentSpanID = "0000000000000000"
	}
	spanDetail.SpanKind = th.getSpanKind(span.Kind)

	attrMap := th.convertKVListToMap(span.Attributes)

	logger.Debug(TRACE_LOG_TAG, "Attribute values ", attrMap)
	if len(span.Events) > 0 {
		for _, event := range span.Events {
			if event.Name == "exception" {
				exceptionDetails := th.createExceptionDetails(event)
				spanIdStr := hex.EncodeToString(span.SpanId)
				hash, err := th.syncExceptionData(exceptionDetails, spanIdStr)
				if err != nil {
					logger.Error(TRACE_LOG_TAG, "Error while syncing exception data for spanId ", spanIdStr, " with error ", err)
				}
				attrMap["exception_hash"] = hash
			}
		}
	}

	if th.otlpConfig.SetSpanAttributes {
		spanDetail.Attributes = attrMap
	}

	sourceIp, destIp := th.getSourceDestIPPair(spanDetail.SpanKind, attrMap, ctx)

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

func (th *TraceHandler) createExceptionDetails(event *tracev1.Span_Event) *model.ExceptionDetails {
	exceptionAttr := event.Attributes
	logger.Debug(TRACE_LOG_TAG, "Exception attributes ", exceptionAttr)
	exception := model.ExceptionDetails{}
	for _, attr := range exceptionAttr {
		switch attr.Key {
		case "exception.stacktrace":
			exception.Stacktrace = attr.Value.GetStringValue()
		case "exception.message":
			exception.Message = attr.Value.GetStringValue()
		case "exception.type":
			exception.Type = attr.Value.GetStringValue()
		}
	}
	return &exception
}

func (th *TraceHandler) syncExceptionData(exception *model.ExceptionDetails, spanId string) (string, error) {
	hash := ""
	if len(exception.Stacktrace) > 0 {
		hash = zkcommon.Generate256SHA(exception.Message, exception.Type, exception.Stacktrace)
		_, ok := th.existingExceptionData.Load(hash)
		if !ok {
			exceptionJSON, err := json.Marshal(exception)
			if err != nil {
				logger.Debug(TRACE_LOG_TAG, "Error encoding exception details for spanID %s: %v\n", spanId, err)
				return "", err
			}
			err = th.exceptionRedisHandler.SetValue(hash, exceptionJSON)
			if err != nil {
				logger.Error(TRACE_LOG_TAG, "Error while saving exception to redis for span Id ", spanId, " with error ", err)
				return "", err
			}
			th.existingExceptionData.Store(hash, true)
		}
	} else {
		logger.Error(TRACE_LOG_TAG, "Could not find stacktrace for exception for span Id ", spanId)
		return "", fmt.Errorf("no stacktrace for the expcetion")
	}
	return hash, nil
}

func (th *TraceHandler) convertKVListToMap(attr []*commonv1.KeyValue) map[string]interface{} {
	attrMap := map[string]interface{}{}
	for _, kv := range attr {
		value := th.getAnyValue(kv.Value)
		if value != nil {
			attrMap[kv.Key] = value
		}
	}
	return attrMap
}

func (th *TraceHandler) getAnyValue(value *commonv1.AnyValue) interface{} {
	switch v := value.Value.(type) {
	case *commonv1.AnyValue_StringValue:
		return v.StringValue
	case *commonv1.AnyValue_ArrayValue:
		var arr []interface{}
		for _, item := range v.ArrayValue.Values {
			arr = append(arr, th.getAnyValue(item))
		}
		return arr
	case *commonv1.AnyValue_BoolValue:
		return v.BoolValue
	case *commonv1.AnyValue_DoubleValue:
		return v.DoubleValue
	case *commonv1.AnyValue_BytesValue:
		return v.BytesValue
	case *commonv1.AnyValue_IntValue:
		return v.IntValue
	default:
		fmt.Println("Variable has an unknown type.")
	}
	return nil
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
