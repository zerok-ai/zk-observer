package main

import (
	"github.com/zerok-ai/zk-otlp-receiver/handler"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"net/http"
)

var LOG_TAG = "main"

func main() {
	http.HandleFunc("/v1/traces", handler.HandleTraceRequest)
	err := http.ListenAndServe(":8147", nil)
	if err != nil {
		logger.Error(LOG_TAG, "Error starting the server:", err)
	}
	logger.Debug(LOG_TAG, "Server started at http://localhost:8147/")
}
