package utils

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/zerok-ai/zk-observer/common"
	"github.com/zerok-ai/zk-observer/model"
	common2 "github.com/zerok-ai/zk-utils-go/common"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/proto/enrichedSpan"
	zkUtilsOtel "github.com/zerok-ai/zk-utils-go/proto/opentelemetry"
	zkmodel "github.com/zerok-ai/zk-utils-go/scenario/model"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"os"
	"sort"
	"strings"
)

var spanUtilsLogTag = "spanUtils"
var NET_SOCK_HOST_ADDR = "net.sock.host.addr"
var NET_SOCK_PEER_ADDR = "net.sock.peer.addr"
var NET_PEER_NAME = "net.peer.name"
var NET_HOST_IP = "net.host.ip"
var NET_PEER_IP = "net.peer.ip"
var SERVER_SOCKET_ADDRESS = "server.socket.address"

const (
	ScopeAttributeHashPrefix    = "sh_" // sh stands for scope hash
	ResourceAttributeHashPrefix = "rh_" // rh stands for resource hash
)

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
			if ok1 && ok2 {
				clientIp, err := GetPodIP(podNameStr, namespaceStr)
				if err == nil {
					sourceIP = clientIp
				}
			}
		}

	} else if spanKind == model.SpanKindServer {
		if len(attributes) > 0 {
			if hostAddr, ok := attributes[NET_SOCK_HOST_ADDR]; ok {
				destIP = hostAddr.(string)
			} else if hostAddr, ok := attributes[SERVER_SOCKET_ADDRESS]; ok {
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
	return enrichedSpan.ConvertKVListToMap(common2.ToPtr(zkUtilsOtel.KeyValueList{KeyValueList: attr}))
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

func GetSchemaVersion(schemaUrl string) string {
	schemaVersion := common.DefaultSchemaVersion
	if len(schemaUrl) > 0 {
		values := strings.Split(schemaUrl, "/")
		if len(values) > 0 {
			schemaVersion = values[len(values)-1]
		}
	}
	return schemaVersion
}

func GetExecutorProtocolFromSpanProtocol(spanProtocol model.ProtocolType) zkmodel.ProtocolName {
	if spanProtocol == model.ProtocolTypeHTTP {
		return zkmodel.ProtocolHTTP
	} else if spanProtocol == model.ProtocolTypeGRPC {
		return zkmodel.ProtocolGRPC
	}
	return zkmodel.ProtocolGeneral
}

func ObjectToInterfaceMap(spanDetails any) map[string]interface{} {
	spanDetailMap := map[string]interface{}{}

	// Json marshal spanDetails.
	spanDetailsJSON, err := json.Marshal(spanDetails)
	if err != nil {
		logger.Error(spanUtilsLogTag, "Error encoding spanDetails ", err)
		return spanDetailMap
	}

	// Json unmarshal spanDetails.
	err = json.Unmarshal(spanDetailsJSON, &spanDetailMap)
	return spanDetailMap
}

func GetResourceIp(spanKind model.SpanKind, sourceIp string, destIp string) string {
	if spanKind == model.SpanKindClient && len(sourceIp) > 0 {
		return sourceIp
	} else if spanKind == model.SpanKindServer && len(destIp) > 0 {
		return destIp
	}
	return ""

}

func GetMD5OfMap(m map[string]interface{}) string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	var mapStr string
	for _, k := range keys {
		mapStr += fmt.Sprintf("%s=%v", k, m[k])
	}

	hash := md5.Sum([]byte(mapStr))
	return hex.EncodeToString(hash[:])
}

func GetServiceName(spanDetailsMap map[string]interface{}) string {
	spanAttributes, ok := spanDetailsMap[common.OTelSpanAttrKey]
	if !ok || spanAttributes == nil {
		logger.Warn(spanUtilsLogTag, "Span attributes not found in spanDetailsMap")
		return common.ScenarioWorkloadGenericServiceNameKey
	}
	spanAttrMap := spanAttributes.(map[string]interface{})
	spanServiceName, nsOk := spanAttrMap[common.OTelSpanAttrServiceNameKey]
	if !nsOk || spanServiceName == "" {
		logger.Warn(spanUtilsLogTag, "Service Name not found in spanAttrMap, using service name='*'. "+
			"Please set OTEL_SERVICE_NAME or `service.name` key in OTEL_RESOURCE_ATTRIBUTES env variable.")
		spanServiceName = common.ScenarioWorkloadGenericServiceNameKey
	}
	return spanServiceName.(string)
}
