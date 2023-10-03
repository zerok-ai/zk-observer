package common

const (
	//Redis DB Names
	FilteredTracesDBName     = "filtered_traces"     // 1
	ScenariosDBName          = "scenarios"           // 2
	TraceDbName              = "traces"              // 3
	ExecutorAttrDBName       = "executor_attr"       // 4
	PodDetailsDbName         = "pod_details"         // 7
	ErrorDetailDBName        = "error_details"       // 8
	IntegrationDetailsDBName = "integration_details" // 9

	//Trace Details Constants
	ParentSpanIdKey         = "parent_span_id"
	SpanKindKey             = "span_kind"
	SourceIpKey             = "source_ip"
	DestIpKey               = "dest_ip"
	ErrorsKey               = "errors"
	StartNsKey              = "start_ns"
	LatencyNsKey            = "latency_ns"
	AttributesKey           = "attributes"
	SatisfiedWorkloadIdsKey = "workload_id_list"

	ResourceLanguageKey = "telemetry.sdk.language"
)
