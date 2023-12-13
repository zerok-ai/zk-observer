package model

import v1 "go.opentelemetry.io/proto/otlp/trace/v1"

type ScopeInfo struct {
	Name          string     `json:"name"`
	Version       string     `json:"version"`
	AttributesMap GenericMap `json:"attributes_map"`
	SchemaUrl     string     `json:"schema_url"`
}

type ResourceInfo struct {
	AttributesMap GenericMap `json:"attributes_map"`
	SchemaUrl     string     `json:"schema_url"`
}

type OtelEnrichedRawSpan struct {
	Span *v1.Span `json:"span"`

	// Span Attributes
	SpanAttributes         GenericMap   `json:"span_attributes,omitempty"`
	SpanEvents             []GenericMap `json:"span_events,omitempty"`
	ResourceAttributesHash string       `json:"resource_attributes_hash,omitempty"`
	ScopeAttributesHash    string       `json:"scope_attributes_hash,omitempty"`

	// ZeroK Properties
	WorkloadIdList []string   `json:"workload_id_list,omitempty"`
	GroupBy        GroupByMap `json:"group_by,omitempty"`
}
