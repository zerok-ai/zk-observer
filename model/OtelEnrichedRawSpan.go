package model

import "github.com/zerok-ai/zk-utils-go/proto/enrichedSpan"

type ScopeInfo struct {
	Name          string                  `json:"name"`
	Version       string                  `json:"version"`
	AttributesMap enrichedSpan.GenericMap `json:"attributes_map"`
	SchemaUrl     string                  `json:"schema_url"`
}

type ResourceInfo struct {
	AttributesMap enrichedSpan.GenericMap `json:"attributes_map"`
	SchemaUrl     string                  `json:"schema_url"`
}
