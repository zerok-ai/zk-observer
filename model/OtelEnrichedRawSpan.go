package model

import zkUtilsCommonModel "github.com/zerok-ai/zk-utils-go/common"

type ScopeInfo struct {
	Name          string                        `json:"name"`
	Version       string                        `json:"version"`
	AttributesMap zkUtilsCommonModel.GenericMap `json:"attributes_map"`
	SchemaUrl     string                        `json:"schema_url"`
}

type ResourceInfo struct {
	AttributesMap zkUtilsCommonModel.GenericMap `json:"attributes_map"`
	SchemaUrl     string                        `json:"schema_url"`
}
