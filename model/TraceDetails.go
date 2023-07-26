package model

import "sync"

type TraceDetails struct {
	SpanDetailsMap sync.Map
}

func (td *TraceDetails) SetSpanDetails(spanID string, spanDetails SpanDetails) {
	td.SpanDetailsMap.Store(spanID, spanDetails)
}
