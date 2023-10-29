package common

const (
	OTelSpanEventException = "exception"

	DefaultParentSpanId = "0000000000000000"

	//Redis DB Names
	FilteredTracesDBName     = "filtered_traces"     // 1
	ScenariosDBName          = "scenarios"           // 2
	TraceDbName              = "traces"              // 3
	ExecutorAttrDBName       = "executor_attr"       // 4
	PodDetailsDbName         = "pod_details"         // 7
	ErrorDetailDBName        = "error_details"       // 8
	IntegrationDetailsDBName = "integration_details" // 9

	ResourceLanguageKey = "telemetry.sdk.language"

	OTelResourceServiceName = "service.name"

	OTelResourceAttrNamespaceKey      = "k8s.namespace.name"
	OTelResourceAttrDeploymentNameKey = "k8s.deployment.name"

	DefaultSchemaVersion = "1.7.0"
)
