package handler

import (
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"io"
	"net"
	"strings"
	"sync"
)

var TRACE_LOG_TAG = "TraceHandler"

var DBSystemAttributeKey = "db.system"
var HTTPMethodAttributeKey = "http.method"
var NetProtocolNameAttributeKey = "net.protocol.name"
var HTTPRouteAttributeKey = "http.route"
var NetPeerNameAttributeKey = "net.peer.name"
var HttpUrlAttributeKey = "http.url"

type TraceHandler struct {
	traceStore   sync.Map
	redisHandler *utils.RedisHandler
	otlpConfig   *config.OtlpConfig
}

func NewTraceHandler(redisHandler *utils.RedisHandler, config *config.OtlpConfig) *TraceHandler {
	handler := TraceHandler{}
	handler.traceStore = sync.Map{}
	handler.redisHandler = redisHandler
	handler.otlpConfig = config
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

// getSourceDestIPPair extracts local and remote IP addresses from spanData and attributes.
func (th *TraceHandler) getSourceDestIPPair(spanKind model.SpanKind, attributes map[string]interface{}) (string, string) {
	destIP := ""
	sourceIP := ""

	if spanKind == model.SpanKindClient {
		ip, err := GetLocalIPAddress()
		if err != nil {
			fmt.Println("Failed to get local IP address")
		} else {
			sourceIP = ip
		}
		if len(attributes) > 0 {
			if peerAddr, ok := attributes["NET_SOCK_PEER_ADDR"]; ok {
				destIP = peerAddr.(string)
			} else if peerName, ok := attributes["NET_PEER_NAME"]; ok {
				address, err := net.LookupHost(peerName.(string))
				if err == nil && len(address) > 0 {
					destIP = address[0]
				}
			}
		}
	} else if spanKind == model.SpanKindServer {
		if len(attributes) > 0 {
			if hostAddr, ok := attributes["NET_SOCK_HOST_ADDR"]; ok {
				destIP = hostAddr.(string)
			}
			if peerAddr, ok := attributes["NET_SOCK_PEER_ADDR"]; ok {
				sourceIP = peerAddr.(string)
			}
		}
	}

	return sourceIP, destIP
}

func GetLocalIPAddress() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String(), nil
		}
	}
	return "", fmt.Errorf("No valid local IP address found")
}

func (th *TraceHandler) createSpanDetails(span *tracev1.Span) model.SpanDetails {
	spanDetail := model.SpanDetails{}
	spanDetail.ParentSpanID = hex.EncodeToString(span.ParentSpanId)
	spanDetail.SpanKind = th.getSpanKind(span.Kind)

	attr := span.Attributes
	attrMap := map[string]interface{}{}
	for _, kv := range attr {
		value := th.getAnyValue(kv.Value)
		if value != nil {
			attrMap[kv.Key] = value
		}
	}

	logger.Debug(TRACE_LOG_TAG, "Attribute values ", attrMap)
	if th.otlpConfig.SetSpanAttributes {
		spanDetail.Attributes = attrMap
	}

	dbSystemAttrValue, dbOk := attrMap[DBSystemAttributeKey]
	httpMethod, httpMethodOk := attrMap[HTTPMethodAttributeKey]
	netProtocolAttrValue, netProtocolOk := attrMap[NetProtocolNameAttributeKey]
	if dbOk {
		tmp := dbSystemAttrValue.(string)
		spanDetail.Protocol = tmp
	} else if httpMethodOk {
		spanDetail.Protocol = "http"
		httpMethodStr := httpMethod.(string)

		if th.otlpConfig.SetHttpEndpoint {
			httpRoute, _ := attrMap["HTTP_ROUTE"]
			netPeerName, _ := attrMap["NET_PEER_NAME"]
			httpURL, _ := attrMap["HTTP_URL"]

			httpRouteStr := ""
			if httpRoute == "" || httpRoute == nil {
				if netPeerName != nil && netPeerName != "" && httpURL != nil && httpURL != "" {
					netPeerNameStr := netPeerName.(string)
					httpURLStr := httpURL.(string)
					httpRouteStr = httpURLStr[strings.Index(httpURLStr, netPeerNameStr)+len(netPeerNameStr):]
				} else {
					httpRouteStr = ""
				}
			} else {
				httpRouteStr = httpRoute.(string)
			}

			spanDetail.Endpoint = "[" + httpMethodStr + "]" + httpRouteStr
		}
	} else if netProtocolOk {
		tmp := netProtocolAttrValue.(string)
		spanDetail.Protocol = tmp
	}

	sourceIp, destIp := th.getSourceDestIPPair(spanDetail.SpanKind, attrMap)

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
