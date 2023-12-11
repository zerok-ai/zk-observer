package common

const (
	OTelSpanEventException = "exception"

	DefaultParentSpanId = "0000000000000000"

	OTelResourceServiceName = "service.name"

	OTelLatencyNsKey    = "latency_ns"
	OTelSpanAttrKey     = "attributes"
	OTelResourceAttrKey = "resource_attributes"
	OTelScopeAttrKey    = "scope_attributes"

	OTelResourceAttrNamespaceKey      = "k8s.namespace.name"
	OTelResourceAttrDeploymentNameKey = "k8s.deployment.name"

	ScenarioWorkloadGenericNamespaceKey  = "*"
	ScenarioWorkloadGenericDeploymentKey = "*"

	DefaultSchemaVersion = "1.7.0"
)
