package common

const (
	OTelSpanEventException = "exception"

	DefaultParentSpanId = "0000000000000000"

	OTelResourceServiceName = "service.name"

	OTelResourceAttrNamespaceKey      = "k8s.namespace.name"
	OTelResourceAttrDeploymentNameKey = "k8s.deployment.name"

	ScenarioWorkloadGenericNamespaceKey  = "*"
	ScenarioWorkloadGenericDeploymentKey = "*"

	DefaultSchemaVersion = "1.7.0"
)
