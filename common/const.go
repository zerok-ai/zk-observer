package common

const (
	OTelSpanEventException = "exception"

	DefaultParentSpanId = "0000000000000000"

	OTelResourceServiceName = "service.name"

	OTelLatencyNsKey     = "latency_ns"
	OTelSpanAttrKey      = "attributes"
	OTelResourceAttrKey  = "resource_attributes"
	OTelScopeAttrKey     = "scope_attributes"
	OTelSchemaVersionKey = "schema_version"
	OTelSpanEventsKey    = "events"
	OTelSpanErrorKey     = "error"

	OTelSpanEventAttrKey          = "attributes"
	OTelSpanEventExceptionHashKey = "exception_hash"

	OTelSpanAttrServiceNameKey        = "service.name"
	OTelResourceAttrNamespaceKey      = "k8s.namespace.name"
	OTelResourceAttrDeploymentNameKey = "k8s.deployment.name"

	ScenarioWorkloadGenericServiceNameKey = "*"
	ScenarioWorkloadGenericNamespaceKey   = "*"
	ScenarioWorkloadGenericDeploymentKey  = "*"

	ServiceListKey = "service_list"

	DefaultSchemaVersion = "1.7.0"
)
