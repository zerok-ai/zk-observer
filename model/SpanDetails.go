package model

type SpanDetails struct {
	SpanKind       SpanKind  `json:"spanKind"`
	ParentSpanID   string    `json:"parentSpanID"`
	RemoteEndpoint *Endpoint `json:"remoteEndpoint,omitempty"`
	LocalEndpoint  *Endpoint `json:"localEndpoint,omitempty"`
	Protocol       string    `json:"protocol,omitempty"`
	Endpoint       string    `json:"endpoint,omitempty"`
	Attributes     string    `json:"attributes,omitempty"`
}
