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

func GetSpanAttributeValue[T string | int | float64 | int64](attributeId AttributeID, spanDetailsMap map[string]interface{}, executorAttrStore stores.ExecutorAttrStore, functionFactory *functions.FunctionFactory) *T {
	if value, ok := evaluator.GetValueFromStore(string(attributeId), spanDetailsMap, functionFactory); ok && value != nil {
		return value.(*T)
	}
	return nil
}
