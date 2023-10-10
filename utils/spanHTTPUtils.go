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
	s.spanDetails.Method = *GetSpanAttributeValue[string](HTTPMethodAttrId, s.spanDetails, s.executorAttrStore)
	s.spanDetails.Route = *GetSpanAttributeValue[string](HTTPRouteAttrId, s.spanDetails, s.executorAttrStore)
	s.spanDetails.Scheme = *GetSpanAttributeValue[string](HTTPSchemeAttrId, s.spanDetails, s.executorAttrStore)
	s.spanDetails.Path = *GetSpanAttributeValue[string](HTTPPathAttrId, s.spanDetails, s.executorAttrStore)
	s.spanDetails.Query = *GetSpanAttributeValue[string](HTTPQueryAttrId, s.spanDetails, s.executorAttrStore)
	s.spanDetails.Status = *GetSpanAttributeValue[int](HTTPStatusAttrId, s.spanDetails, s.executorAttrStore)
	s.spanDetails.Username = *GetSpanAttributeValue[string](HTTPUsernameAttrId, s.spanDetails, s.executorAttrStore)
}
