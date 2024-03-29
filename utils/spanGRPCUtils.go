package utils

// Ref: https://docs.google.com/spreadsheets/d/1E_MoV1mRL96hdTv2Q0o3pIAQ1hRF3-RQyF56kLagF04/edit#gid=1422911777
const (
	GRPCIdentifierAttrId AttributeID = "rpc_identifier"

	GRPCSystemAttrId   AttributeID = "rpc_system"
	GRPCMethodAttrId   AttributeID = "rpc_method"
	GRPCRouteAttrId    AttributeID = "rpc_service"
	GRPCSchemeAttrId   AttributeID = "rpc_system"
	GRPCPathAttrId     AttributeID = "rpc_service"
	GRPCQueryAttrId    AttributeID = ""
	GRPCStatusAttrId   AttributeID = "rpc_grpc_status_code"
	GRPCUsernameAttrId AttributeID = ""
)

func (s SpanProtocolUtil) AddGRPCSpanProperties() {
	s.spanDetails.Method = GetSpanAttributeValue[string](GRPCMethodAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Route = GetSpanAttributeValue[string](GRPCRouteAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Scheme = GetSpanAttributeValue[string](GRPCSchemeAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Path = GetSpanAttributeValue[string](GRPCPathAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Query = GetSpanAttributeValue[string](GRPCQueryAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Status = GetSpanAttributeValue[float64](GRPCStatusAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Username = GetSpanAttributeValue[string](GRPCUsernameAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
}
