package utils

import (
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
)

type AttributeID string

// DetectSpanProtocolMap Mapping of span attributes to protocol type.
var DetectSpanProtocolMap = map[AttributeID]model.ProtocolType{
	HTTPMethodAttrId: model.ProtocolTypeHTTP,
}

type SpanProtocolUtil struct {
	spanDetails       model.OTelSpanDetails
	executorAttrStore stores.ExecutorAttrStore
}

func NewSpanProtocolUtil(spanDetails model.OTelSpanDetails, executorAttrStore stores.ExecutorAttrStore) SpanProtocolUtil {
	return SpanProtocolUtil{
		spanDetails:       spanDetails,
		executorAttrStore: executorAttrStore,
	}
}

func (s SpanProtocolUtil) DetectSpanProtocol() model.ProtocolType {
	for attributeId, protocol := range DetectSpanProtocolMap {
		if val := GetSpanAttributeValue[string](attributeId, s.spanDetails, s.executorAttrStore); val != nil {
			return protocol
		}
	}
	return model.ProtocolTypeUnknown
}

func (s SpanProtocolUtil) AddSpanProtocolProperties() {
	if s.spanDetails.Protocol == model.ProtocolTypeHTTP {
		s.AddHTTPSpanProperties()
	}
}
