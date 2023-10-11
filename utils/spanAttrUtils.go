package utils

import (
	OTlpSpanModel "github.com/zerok-ai/zk-otlp-receiver/model"
	ExecutorModel "github.com/zerok-ai/zk-utils-go/scenario/model"
	evaluator "github.com/zerok-ai/zk-utils-go/scenario/model/evaluators"
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

func GetAttributePath(attributeId AttributeID, spanDetailsMap map[string]interface{}, executorAttrStore stores.ExecutorAttrStore) string {
	schemaVersion := GetSchemaVersionFromSpanDetailsMap(spanDetailsMap)
	attributePath := *executorAttrStore.Get(ExecutorModel.ExecutorOTel, schemaVersion, ExecutorModel.ProtocolHTTP, string(attributeId))
	return attributePath
}

func GetSpanAttributeValue[T string | int | float64 | int64](attrId AttributeID, spanDetailsMap *map[string]interface{}, executorAttrStore stores.ExecutorAttrStore, functionFactory *functions.FunctionFactory) *T {
	attrPath := GetAttributePath(attrId, *spanDetailsMap, executorAttrStore)
	if value, ok := evaluator.GetValueFromStore(attrPath, *spanDetailsMap, functionFactory); ok && value != nil {
		return value.(*T)
	}
	return nil
}
