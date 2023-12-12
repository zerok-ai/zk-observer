package model

import v1 "go.opentelemetry.io/proto/otlp/trace/v1"

type OtelEnrichedRawSpan struct {
	Span *v1.Span `json:"-"`

	// Span Attributes
	ResourceAttributesHash string `json:"resource_attributes_hash,omitempty"`
	ScopeAttributesHash    string `json:"scope_attributes_hash,omitempty"`

	// ZeroK Properties
	WorkloadIdList []string   `json:"workload_id_list,omitempty"`
	GroupBy        GroupByMap `json:"group_by,omitempty"`
}
