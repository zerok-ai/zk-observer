package model

// SpanKind represents the type of span.
type SpanKind string

// Type of Span. Can be used to specify additional relationships between spans in addition to a parent/child relationship.
const (

	// SpanKindInternal Default value. Indicates that the span is used internally.
	SpanKindInternal SpanKind = "INTERNAL"

	// SpanKindServer Indicates that the span covers server-side handling of an RPC or other remote request.
	SpanKindServer SpanKind = "SERVER"

	// SpanKindClient Indicates that the span covers the client-side wrapper around an RPC or other remote request.
	SpanKindClient SpanKind = "CLIENT"

	// SpanKindProducer Indicates that the span describes producer sending a message to a broker.
	//Unlike client and server, there is no direct critical path latency relationship between producer and consumer spans.
	SpanKindProducer SpanKind = "PRODUCER"

	// SpanKindConsumer Indicates that the span describes consumer receiving a message from a broker.
	//Unlike client and server, there is no direct critical path latency relationship between producer and consumer spans
	SpanKindConsumer SpanKind = "CONSUMER"
)
