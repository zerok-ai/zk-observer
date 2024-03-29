package model

import (
	"github.com/zerok-ai/zk-observer/common"
	zkUtilsCommonModel "github.com/zerok-ai/zk-utils-go/common"
)

func GenericMapPtrFromMap(inputMap map[string]interface{}) *zkUtilsCommonModel.GenericMap {
	genericMap := zkUtilsCommonModel.GenericMap(inputMap)
	return &genericMap
}

type OTelSpanDetails struct {
	// Span common Properties
	ParentSpanId  string          `json:"parent_span_id"`
	SpanKind      SpanKind        `json:"span_kind"`
	StartNs       uint64          `json:"start_ns"`
	LatencyNs     uint64          `json:"latency_ns"`
	SchemaVersion string          `json:"schema_version"`
	Errors        []SpanErrorInfo `json:"errors,omitempty"`

	// Span Attributes
	SpanAttributes     *zkUtilsCommonModel.GenericMap `json:"attributes,omitempty"`
	ResourceAttributes *zkUtilsCommonModel.GenericMap `json:"resource_attributes,omitempty"`
	ScopeAttributes    *zkUtilsCommonModel.GenericMap `json:"scope_attributes,omitempty"`

	ResourceAttributesHash string `json:"resource_attributes_hash,omitempty"`
	ScopeAttributesHash    string `json:"scope_attributes_hash,omitempty"`

	// Span Identifier Properties
	ServiceName string `json:"service_name"`
	SpanName    string `json:"span_name"`

	Protocol ProtocolType `json:"protocol"`

	// Network span properties
	SourceIp    *string `json:"source_ip,omitempty"`
	Source      *string `json:"source,omitempty"`
	DestIp      *string `json:"destination_ip,omitempty"`
	Destination *string `json:"destination,omitempty"`

	// Protocol properties.
	Method   *string  `json:"method,omitempty"`
	Route    *string  `json:"route,omitempty"`
	Scheme   *string  `json:"scheme,omitempty"`
	Path     *string  `json:"path,omitempty"`
	Query    *string  `json:"query,omitempty"`
	Status   *float64 `json:"status,omitempty"`
	Username *string  `json:"username,omitempty"`

	// ZeroK Properties
	WorkloadIdList []string                      `json:"workload_id_list,omitempty"`
	GroupBy        zkUtilsCommonModel.GroupByMap `json:"group_by,omitempty"`
}

type SpanErrorInfo struct {
	Message       string    `json:"message"`
	ErrorType     ErrorType `json:"error_type"`
	ExceptionType string    `json:"exception_type"`
	Hash          string    `json:"hash"`
}

type ErrorType string

const (
	ErrorTypeException ErrorType = "exception"
)

type ProtocolType string

const (
	ProtocolTypeHTTP    ProtocolType = "HTTP"
	ProtocolTypeDB      ProtocolType = "DB"
	ProtocolTypeGRPC    ProtocolType = "GRPC"
	ProtocolTypeUnknown ProtocolType = "UNKNOWN"
)

func (s *OTelSpanDetails) SetParentSpanId(parentSpanId string) {
	if len(parentSpanId) == 0 {
		parentSpanId = common.DefaultParentSpanId
	}
	s.ParentSpanId = parentSpanId
}

func (s *OTelSpanDetails) GetResourceIP() string {
	spanKind := s.SpanKind
	if spanKind == SpanKindClient && s.SourceIp != nil {
		return *s.SourceIp
	} else if spanKind == SpanKindServer && s.DestIp != nil {
		return *s.DestIp
	}
	return ""
}
