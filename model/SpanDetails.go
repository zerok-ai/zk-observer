package model

type SpanDetails struct {
	SpanKind       SpanKind `json:"spanKind"`
	ParentSpanID   string   `json:"parentSpanID"`
	RemoteEndpoint Endpoint `json:"remoteEndpoint"`
	LocalEndpoint  Endpoint `json:"localEndpoint"`
	Protocol       string   `json:"protocol"`
	Endpoint       string   `json:"endpoint"`
	Attributes     string   `json:"attributes"`
}
