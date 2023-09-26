package utils

import (
	"context"
	"encoding/json"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"time"
)

var traceDbName = "traces"
var traceRedisHandlerLogTag = "TraceRedisHandler"

type TraceRedisHandler struct {
	redisHandler *RedisHandler
	ctx          context.Context
	config       *config.OtlpConfig
}

func NewTracesRedisHandler(otlpConfig *config.OtlpConfig) (*TraceRedisHandler, error) {
	redisHandler, err := NewRedisHandler(&otlpConfig.Redis, traceDbName, otlpConfig.Traces.SyncDuration, otlpConfig.Traces.BatchSize, traceRedisHandlerLogTag)

	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while creating redis client ", err)
	}

	handler := &TraceRedisHandler{
		redisHandler: redisHandler,
		ctx:          context.Background(),
		config:       otlpConfig,
	}

	return handler, nil
}

func (h *TraceRedisHandler) PutTraceData(traceId string, spanId string, spanDetails map[string]interface{}) error {

	err := h.redisHandler.CheckRedisConnection()
	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while checking redis conn ", err)
		return err
	}
	spanJsonMap := make(map[string]interface{})
	spanJSON, err := json.Marshal(spanDetails)
	if err != nil {
		logger.Debug(traceRedisHandlerLogTag, "Error encoding SpanDetails for spanID %s: %v\n", spanId, err)
		return err
	}
	spanJsonMap[spanId] = string(spanJSON)
	err = h.redisHandler.HMSetPipeline(traceId, spanJsonMap, time.Duration(h.config.Traces.Ttl)*time.Second)
	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while setting trace details for traceId %s: %v\n", traceId, err)
		return err
	}
	return nil
}
