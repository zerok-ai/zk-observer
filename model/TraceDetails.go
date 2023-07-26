package model

type TraceDetails struct {
	SpanDetailsMap map[string]SpanDetails
}

func (td *TraceDetails) SetSpanDetails(spanID string, spanDetails SpanDetails) {
	if td.SpanDetailsMap == nil {
		td.SpanDetailsMap = make(map[string]SpanDetails)
	}
	td.SpanDetailsMap[spanID] = spanDetails
}

func (td *TraceDetails) GetSpanDetails(spanID string) SpanDetails {
	return td.SpanDetailsMap[spanID]
}
