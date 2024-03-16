package utils

import "fmt"

type TelemetryRequestType string

const (
	INGRESS_TO_EXTERNAL TelemetryRequestType = "ie"
	EXTERNAL_TO_INGRESS TelemetryRequestType = "ei"
	INGRESS_TO_PODS     TelemetryRequestType = "ip"	
	PODS_TO_INGRESS     TelemetryRequestType = "pi"
)


func BuildCollectionName(baseName string, reqType TelemetryRequestType) string {
	return fmt.Sprintf("%s-%s", baseName, string(reqType))
}

