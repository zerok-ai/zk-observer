package model

type SpanDetails struct {
	SpanKind     SpanKind               `json:"span_kind"`
	ParentSpanID string                 `json:"parent_span_id"`
	Endpoint     string                 `json:"endpoint,omitempty"`
	ResourceAttr map[string]interface{} `json:"resource_attributes,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	StartNs      uint64                 `json:"start_ns"`
	EndNs        uint64                 `json:"end_ns"`
	SourceIP     string                 `json:"source_ip,omitempty"`
	DestIP       string                 `json:"dest_ip,omitempty"`
}
