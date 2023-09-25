package model

type SpanDetails struct {
	SpanKind     SpanKind               `json:"span_kind"`
	ParentSpanID string                 `json:"parent_span_id"`
	Endpoint     string                 `json:"endpoint,omitempty"`
	Errors       []interface{}          `json:"errors,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	StartNs      uint64                 `json:"start_ns"`
	LatencyNs    uint64                 `json:"latency_ns"`
	SourceIP     string                 `json:"source_ip,omitempty"`
	DestIP       string                 `json:"dest_ip,omitempty"`
}

type SpanDetailsException struct {
	ErrorType     string `json:"error_type"`
	Hash          string `json:"hash"`
	ExceptionType string `json:"exception_type"`
	Message       string `json:"message"`
}
