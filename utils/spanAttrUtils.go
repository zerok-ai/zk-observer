package utils

import (
	OTlpSpanModel "github.com/zerok-ai/zk-otlp-receiver/model"
	ExecutorModel "github.com/zerok-ai/zk-utils-go/scenario/model"
	"github.com/zerok-ai/zk-utils-go/scenario/model/evaluators/cache"
	"github.com/zerok-ai/zk-utils-go/scenario/model/evaluators/functions"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
)

func getExecutorAttrProtocol(protocolType OTlpSpanModel.ProtocolType) ExecutorModel.ProtocolName {
	if protocolType == OTlpSpanModel.ProtocolTypeHTTP {
		return ExecutorModel.ProtocolHTTP
	}
	return ExecutorModel.ProtocolGeneral
}

func GetSchemaVersionFromSpanDetailsMap(spanDetailsMap map[string]interface{}) string {
	schemaVersion, _ := spanDetailsMap["schema_version"].(string)
	return schemaVersion
}

func GenerateAttribStoreKey(spanDetailsMap map[string]interface{}, protocol ExecutorModel.ProtocolName) cache.AttribStoreKey {
	schemaVersion := GetSchemaVersionFromSpanDetailsMap(spanDetailsMap)
	attribKey, _ := cache.CreateKey(ExecutorModel.ExecutorOTel, schemaVersion, protocol)
	return attribKey
}

func GetSpanAttributeValue[T string | float64](attrId AttributeID, spanDetailsMap *map[string]interface{}, executorAttrStore *stores.ExecutorAttrStore, functionFactory *functions.FunctionFactory, attribStoreKey *cache.AttribStoreKey) *T {
	if attrId != "" {
		if value, ok := functionFactory.EvaluateString(string(attrId), *spanDetailsMap, attribStoreKey); ok && value != nil {
			var x = value.(T)
			return &x
		}
	}
	return nil
}
