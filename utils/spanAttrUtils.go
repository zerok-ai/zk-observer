package utils

import (
	OTlpSpanModel "github.com/zerok-ai/zk-otlp-receiver/model"
	ExecutorModel "github.com/zerok-ai/zk-utils-go/scenario/model"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
)

func getExecutorAttrProtocol(protocolType OTlpSpanModel.ProtocolType) ExecutorModel.ProtocolName {
	if protocolType == OTlpSpanModel.ProtocolTypeHTTP {
		return ExecutorModel.ProtocolHTTP
	}
	return ExecutorModel.ProtocolGeneral
}

func GetSpanAttributeValue[T string | int | float64 | int64](attributeId AttributeID, spanDetails OTlpSpanModel.OTelSpanDetails, executorAttrStore stores.ExecutorAttrStore) *T {
	attributeStoreProtocol := getExecutorAttrProtocol(spanDetails.Protocol)
	key := executorAttrStore.Get(ExecutorModel.ExecutorOTel, spanDetails.SchemaVersion, attributeStoreProtocol, string(attributeId))
	if key == nil {
		return nil
	}
	var value, ok = spanDetails.Attributes[*key]
	if ok {
		val := value.(T)
		return &val
	}
	return nil
}
