package utils

import (
	"context"
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"os"
)

var spanUtilsLogTag = "spanUtils"
var NET_SOCK_HOST_ADDR = "net.sock.host.addr"
var NET_SOCK_PEER_ADDR = "net.sock.peer.addr"
var NET_PEER_NAME = "net.peer.name"
var NET_HOST_IP = "net.host.ip"
var NET_PEER_IP = "net.peer.ip"

func GetSourceDestIPPair(spanKind model.SpanKind, attributes map[string]interface{}, resourceAttrMap map[string]interface{}) (string, string) {
	destIP := ""
	sourceIP := ""

	if spanKind == model.SpanKindClient {
		if len(attributes) > 0 {
			if peerAddr, ok := attributes[NET_SOCK_PEER_ADDR]; ok {
				destIP = peerAddr.(string)
			} else if peerName, ok := attributes[NET_PEER_NAME]; ok {
				address, err := net.LookupHost(peerName.(string))
				if err == nil && len(address) > 0 {
					destIP = address[0]
				}
			}
		}
		podName, ok1 := resourceAttrMap["k8s.pod.name"]
		namespace, ok2 := resourceAttrMap["k8s.namespace.name"]
		if ok1 && ok2 {
			podNameStr, ok1 := podName.(string)
			namespaceStr, ok2 := namespace.(string)
			logger.Debug(spanUtilsLogTag, "Pod name is ", podNameStr, " namespace is ", namespaceStr)
			if ok1 && ok2 {
				clientIp, err := GetPodIP(podNameStr, namespaceStr)
				//logger.Debug(spanUtilsLogTag, "Client IP is ", clientIp, " error is ", err)
				if err == nil {
					sourceIP = clientIp
				}
			}
		}

	} else if spanKind == model.SpanKindServer {
		if len(attributes) > 0 {
			if hostAddr, ok := attributes[NET_SOCK_HOST_ADDR]; ok {
				destIP = hostAddr.(string)
			} else if hostAddr, ok := attributes[NET_HOST_IP]; ok {
				destIP = hostAddr.(string)
			}

			if peerAddr, ok := attributes[NET_SOCK_PEER_ADDR]; ok {
				sourceIP = peerAddr.(string)
			} else if peerAddr, ok := attributes[NET_PEER_IP]; ok {
				sourceIP = peerAddr.(string)
			}
		}
	}

	if len(destIP) > 0 {
		destIP = ConvertToIpv4(destIP)
	}

	if len(sourceIP) > 0 {
		sourceIP = ConvertToIpv4(sourceIP)
	}

	return sourceIP, destIP
}

func ConvertToIpv4(ipStr string) string {
	ip := net.ParseIP(ipStr)

	if ip == nil {
		logger.Error(spanUtilsLogTag, "Invalid IP address ", ipStr)
		return ""
	}

	if ip.To4() != nil {
		ipv4 := ip.String()
		return ipv4
	}
	return ""
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
		logger.Debug(spanUtilsLogTag, "Unknown type ", v)
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

func GetPodIP(podName string, namespace string) (string, error) {
	clientset, err := GetK8sClient()
	if err != nil {
		return "", err
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return pod.Status.PodIP, nil
}

func GetK8sClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// If incluster config failes, reading from kubeconfig.
		// However, this is not connecting to gcp clusters. Only working for kind now(probably minikube also).
		kubeconfig := os.Getenv("KUBECONFIG")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes config: %v", err)
		}
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
