package utils

import (
	OTlpSpanModel "github.com/zerok-ai/zk-otlp-receiver/model"
	ExecutorModel "github.com/zerok-ai/zk-utils-go/scenario/model"
	evaluator "github.com/zerok-ai/zk-utils-go/scenario/model/evaluators"
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

func GenerateAttribStoreKey(spanDetailsMap map[string]interface{}) cache.AttribStoreKey {
	schemaVersion := GetSchemaVersionFromSpanDetailsMap(spanDetailsMap)
	attribKey, _ := cache.CreateKey(ExecutorModel.ExecutorOTel, schemaVersion, ExecutorModel.ProtocolHTTP)
	return attribKey
}

func GetAttributePath(attributeId AttributeID, spanDetailsMap map[string]interface{}, executorAttrStore *stores.ExecutorAttrStore) string {
	attribKey := GenerateAttribStoreKey(spanDetailsMap)
	attributePath := *executorAttrStore.GetAttributeFromStore(attribKey, string(attributeId))
	return attributePath
}

func GetSpanAttributeValue[T string | float64](attrId AttributeID, spanDetailsMap *map[string]interface{}, executorAttrStore *stores.ExecutorAttrStore, functionFactory *functions.FunctionFactory, attribStoreKey *cache.AttribStoreKey) *T {
	if attrId != "" {
		if value, ok := evaluator.GetValueFromStore(string(attrId), *spanDetailsMap, functionFactory, attribStoreKey); ok && value != nil {
			var x = value.(T)
			return &x
		}
	}
	var x T
	return &x
}
