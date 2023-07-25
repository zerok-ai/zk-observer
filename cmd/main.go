package main

import (
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/handler"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"net/http"
)

var LOG_TAG = "main"

func main() {

	otlpConfig := config.CreateConfig()
	logger.Init(otlpConfig.LogsConfig)

	http.HandleFunc("/v1/traces", handler.HandleTraceRequest)

	logger.Debug(LOG_TAG, "Starting Server at http://localhost:8147/")

	err := http.ListenAndServe(":8147", nil)
	if err != nil {
		logger.Error(LOG_TAG, "Error starting the server:", err)
	}

}
