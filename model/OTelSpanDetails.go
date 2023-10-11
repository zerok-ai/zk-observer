package model

import (
	"github.com/zerok-ai/zk-otlp-receiver/common"
)

type OTelSpanDetails struct {
	// Span common Properties
	TraceId       string                 `json:"trace_id"`
	SpanId        string                 `json:"span_id"`
	ParentSpanId  string                 `json:"parent_span_id"`
	SpanKind      SpanKind               `json:"span_kind"`
	StartNs       uint64                 `json:"start_ns"`
	LatencyNs     uint64                 `json:"latency_ns"`
	SchemaVersion string                 `json:"schema_version"`
	SourceIp      string                 `json:"source_ip"`
	Source        string                 `json:"source"`
	DestIp        string                 `json:"dest_ip"`
	Destination   string                 `json:"destination"`
	Attributes    map[string]interface{} `json:"attributes"`
	Errors        []SpanErrorInfo        `json:"errors"`

	Protocol ProtocolType `json:"protocol"`

	// Protocol properties.
	Method   string  `json:"method"`
	Route    string  `json:"route"`
	Scheme   string  `json:"scheme"`
	Path     string  `json:"path"`
	Query    string  `json:"query"`
	Status   float64 `json:"status"`
	Username string  `json:"username"`

	// ZeroK Properties
	WorkloadIdList []string        `json:"workload_id_list"`
	GroupBy        []GroupByValues `json:"group_by"`
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
	ProtocolTypeUnknown ProtocolType = "UNKNOWN"
)

type GroupByValues struct {
	WorkloadId string `json:"workload_id"`
	Title      string `json:"title"`
	Hash       string `json:"hash"`
}

func (s OTelSpanDetails) SetParentSpanId(parentSpanId string) {
	if len(parentSpanId) == 0 {
		parentSpanId = common.DefaultParentSpanId
	}
	s.ParentSpanId = parentSpanId
}

func (s OTelSpanDetails) GetResourceIP() string {
	spanKind := s.SpanKind
	if spanKind == SpanKindClient {
		return s.SourceIp
	} else if spanKind == SpanKindServer {
		return s.DestIp
	}
	return ""
}
