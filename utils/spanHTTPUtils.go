package utils

const (
	HTTPMethodAttrId   AttributeID = "http_request_method"
	HTTPRouteAttrId    AttributeID = "http_route"
	HTTPSchemeAttrId   AttributeID = "url_scheme"
	HTTPPathAttrId     AttributeID = "url_full"
	HTTPQueryAttrId    AttributeID = ""
	HTTPStatusAttrId   AttributeID = "http_response_status_code"
	HTTPUsernameAttrId AttributeID = ""
)

func (s SpanProtocolUtil) AddHTTPSpanProperties() {
	s.spanDetails.Method = GetSpanAttributeValue[string](HTTPMethodAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory)
	s.spanDetails.Route = GetSpanAttributeValue[string](HTTPRouteAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory)
	s.spanDetails.Scheme = GetSpanAttributeValue[string](HTTPSchemeAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory)
	s.spanDetails.Path = GetSpanAttributeValue[string](HTTPPathAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory)
	s.spanDetails.Query = GetSpanAttributeValue[string](HTTPQueryAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory)
	s.spanDetails.Status = GetSpanAttributeValue[float64](HTTPStatusAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory)
	s.spanDetails.Username = GetSpanAttributeValue[string](HTTPUsernameAttrId, s.spanDetailsMap, s.executorAttrStore, s.functionFactory)
}
