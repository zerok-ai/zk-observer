package utils

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
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
	pipeline     redis.Pipeliner
}

func NewTracesRedisHandler(otlpConfig *config.OtlpConfig) (*TraceRedisHandler, error) {
	redisHandler, err := NewRedisHandler(&otlpConfig.Redis, traceDbName, otlpConfig.Traces.SyncDuration, otlpConfig.Traces.BatchSize)

	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while creating redis client ", err)
	}

	handler := &TraceRedisHandler{
		redisHandler: redisHandler,
		ctx:          context.Background(),
		config:       otlpConfig,
	}
	handler.pipeline = redisHandler.Pipeline

	return handler, nil
}

func (h *TraceRedisHandler) PutTraceData(traceId string, spanId string, spanDetails map[string]interface{}) error {

	err := h.redisHandler.CheckRedisConnection()
	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while checking redis conn ", err)
	}
	spanJsonMap := make(map[string]string)
	spanJSON, err := json.Marshal(spanDetails)
	if err != nil {
		logger.Debug(traceRedisHandlerLogTag, "Error encoding SpanDetails for spanID %s: %v\n", spanId, err)
		return err
	}
	spanJsonMap[spanId] = string(spanJSON)

	ctx := context.Background()
	//logger.Debug(traceRedisHandlerLogTag, "Len of redis pipeline ", h.pipeline.Len())
	h.pipeline.HMSet(ctx, traceId, spanJsonMap)
	h.pipeline.Expire(ctx, traceId, time.Duration(h.config.Traces.Ttl)*time.Second)
	h.count++
	h.syncPipeline()
	return nil
}
