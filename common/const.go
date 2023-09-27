package common

var (
	RedisScenarioDbName = "scenario_db"
	WorkloadSpanDbName  = "workloads"

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
