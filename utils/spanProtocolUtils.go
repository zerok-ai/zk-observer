package utils

import (
	"github.com/zerok-ai/zk-observer/model"
	"github.com/zerok-ai/zk-utils-go/scenario/model/evaluators/cache"
	"github.com/zerok-ai/zk-utils-go/scenario/model/evaluators/functions"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
)

type AttributeID string

// DetectSpanProtocolMap Mapping of span attributes to protocol type.
var DetectSpanProtocolMap = map[AttributeID]model.ProtocolType{
	HTTPIdentifierAttrId: model.ProtocolTypeHTTP,
	DBIdentifierAttrId:   model.ProtocolTypeDB,
	GRPCIdentifierAttrId: model.ProtocolTypeGRPC,
}

type SpanProtocolUtil struct {
	spanDetails       *model.OTelSpanDetails
	spanDetailsMap    *map[string]interface{}
	executorAttrStore *stores.ExecutorAttrStore
	podDetailsStore   *stores.LocalCacheHSetStore
	attrStoreKey      *cache.AttribStoreKey
	functionFactory   *functions.FunctionFactory
}

func NewSpanProtocolUtil(spanDetails *model.OTelSpanDetails, spanDetailsMap *map[string]interface{}, executorAttrStore *stores.ExecutorAttrStore, podDetailsStore *stores.LocalCacheHSetStore, attrStoreKey *cache.AttribStoreKey) SpanProtocolUtil {
	ff := functions.NewFunctionFactory(podDetailsStore, executorAttrStore)
	return SpanProtocolUtil{
		spanDetails:       spanDetails,
		spanDetailsMap:    spanDetailsMap,
		executorAttrStore: executorAttrStore,
		functionFactory:   ff,
		attrStoreKey:      attrStoreKey,
	}
}

func (s SpanProtocolUtil) DetectSpanProtocol() model.ProtocolType {
	for attributeId, protocol := range DetectSpanProtocolMap {
		if val, ok := s.functionFactory.EvaluateString(string(attributeId), *s.spanDetailsMap, s.attrStoreKey); ok && val != nil {
			return protocol
		}
	}
	return model.ProtocolTypeUnknown
}

func (s SpanProtocolUtil) AddSpanProtocolProperties() {
	if s.spanDetails.Protocol == model.ProtocolTypeHTTP {
		s.AddHTTPSpanProperties()
	} else if s.spanDetails.Protocol == model.ProtocolTypeGRPC {
		s.AddGRPCSpanProperties()
	} else if s.spanDetails.Protocol == model.ProtocolTypeDB {
		/* TODO: Implement following method with DB attributes. */
		// s.AddDBSpanProperties()
	}
}
