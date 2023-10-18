package utils

// Ref: https://docs.google.com/spreadsheets/d/1E_MoV1mRL96hdTv2Q0o3pIAQ1hRF3-RQyF56kLagF04/edit#gid=1422911777
const (
	DBIdentifierAttrId AttributeID = "db_identifier"

	DBSystemAttrId   AttributeID = "db_system"
	DBMethodAttrId   AttributeID = ""
	DBRouteAttrId    AttributeID = ""
	DBSchemeAttrId   AttributeID = ""
	DBPathAttrId     AttributeID = ""
	DBQueryAttrId    AttributeID = ""
	DBStatusAttrId   AttributeID = ""
	DBUsernameAttrId AttributeID = ""
)

func (s SpanProtocolUtil) AddDBSpanProperties() {
	s.spanDetails.Method = GetSpanAttributeValue[string](DBMethodAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Route = GetSpanAttributeValue[string](DBRouteAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Scheme = GetSpanAttributeValue[string](DBSchemeAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Path = GetSpanAttributeValue[string](DBPathAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Query = GetSpanAttributeValue[string](DBQueryAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Status = GetSpanAttributeValue[float64](DBStatusAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
	s.spanDetails.Username = GetSpanAttributeValue[string](DBUsernameAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory, s.attrStoreKey)
}
