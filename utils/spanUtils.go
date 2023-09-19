package utils

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"net"
	"net/http"
)

var UTILS_LOG_TAG = "utils"
var NET_SOCK_HOST_ADDR = "net.sock.host.addr"
var NET_SOCK_PEER_ADDR = "net.sock.peer.addr"
var NET_PEER_NAME = "net.peer.name"

func GetClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return host
}

func GetSourceDestIPPair(spanKind model.SpanKind, attributes map[string]interface{}, ctx iris.Context) (string, string) {
	destIP := ""
	sourceIP := ""

	if spanKind == model.SpanKindClient {
		if len(attributes) > 0 {
			sourceIP = GetClientIP(ctx.Request())
			logger.Debug(UTILS_LOG_TAG, "Source Ip for client span  is ", sourceIP)
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

func ConvertKVListToMap(attr []*commonv1.KeyValue) map[string]interface{} {
	attrMap := map[string]interface{}{}
	for _, kv := range attr {
		value := GetAnyValue(kv.Value)
		if value != nil {
			attrMap[kv.Key] = value
		}
	}
	return attrMap
}

func GetAnyValue(value *commonv1.AnyValue) interface{} {
	switch v := value.Value.(type) {
	case *commonv1.AnyValue_StringValue:
		return v.StringValue
	case *commonv1.AnyValue_ArrayValue:
		var arr []interface{}
		for _, item := range v.ArrayValue.Values {
			arr = append(arr, GetAnyValue(item))
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

func GetSpanKind(kind tracev1.Span_SpanKind) model.SpanKind {
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