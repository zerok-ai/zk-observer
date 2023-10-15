package model

var TelemetryAutoVersionKey = "telemetry.auto.version"
var TelemetrySdkLanguageKey = "telemetry.sdk.language"
var TelemetrySdkNameKey = "telemetry.sdk.name"
var TelemetrySdkVersionKey = "telemetry.sdk.version"
var ServiceNameKey = "service.name"
var ServiceVersionKey = "service.version"

type TelemetryDetails struct {
	TelemetryAutoVersion string `json:"telemetry_auto_version"`
	TelemetrySdkLanguage string `json:"telemetry_sdk_language"`
	TelemetrySdkName     string `json:"telemetry_sdk_name"`
	TelemetrySdkVersion  string `json:"telemetry_sdk_version"`
	ServiceName          string `json:"service_name"`
	ServiceVersion       string `json:"service_version"`
}

func CreateTelemetryDetails(resourceAttrMap map[string]interface{}) *TelemetryDetails {
	telemetryDetails := TelemetryDetails{}
	telemetryDetails.TelemetryAutoVersion = getStringFromMap(resourceAttrMap, TelemetryAutoVersionKey)
	telemetryDetails.TelemetrySdkLanguage = getStringFromMap(resourceAttrMap, TelemetrySdkLanguageKey)
	telemetryDetails.TelemetrySdkName = getStringFromMap(resourceAttrMap, TelemetrySdkNameKey)
	telemetryDetails.TelemetrySdkVersion = getStringFromMap(resourceAttrMap, TelemetrySdkVersionKey)
	telemetryDetails.ServiceName = getStringFromMap(resourceAttrMap, ServiceNameKey)
	telemetryDetails.ServiceVersion = getStringFromMap(resourceAttrMap, ServiceVersionKey)
	return &telemetryDetails
}

func getStringFromMap(resourceAttrMap map[string]interface{}, key string) string {
	value, ok := resourceAttrMap[key]
	if ok {
		return value.(string)
	}
	return ""
}
