package model

type ExceptionDetails struct {
	Message    string `json:"message"`
	Stacktrace string `json:"stacktrace"`
	Type       string `json:"type"`
}
