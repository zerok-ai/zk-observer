package main

import (
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/handler"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"net/http"
)

var LOG_TAG = "main"

func main() {

	otlpConfig := config.CreateConfig()

	if err := utils.ProcessArgs(otlpConfig); err != nil {
		logger.Error(LOG_TAG, "Unable to process wsp client config. Stopping wsp client.")
		return
	}

	logger.Init(otlpConfig.Logs)

	redisHandler, err := utils.NewRedisHandler(&otlpConfig.Redis)
	if err != nil {
		logger.Error(LOG_TAG, "Error while creating redis handler:", err)
		return
	}

	traceHandler := handler.NewTraceHandler(redisHandler)

	http.HandleFunc("/v1/traces", traceHandler.HandleTraceRequest)

	logger.Debug(LOG_TAG, "Starting Server at http://localhost:8147/")

	err = http.ListenAndServe(":8147", nil)
	if err != nil {
		logger.Error(LOG_TAG, "Error starting the server:", err)
	}

}
