package redis

import (
	"context"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/storage/redis/clientDBNames"
	"os"
	"time"
)

var traceRedisHandlerLogTag = "TraceRedisHandler"

type TraceRedisHandler struct {
	redisHandler *RedisHandler
	ctx          context.Context
	config       *config.OtlpConfig
	nodeIP       string
}

func NewTracesRedisHandler(otlpConfig *config.OtlpConfig) (*TraceRedisHandler, error) {
	redisHandler, err := NewRedisHandler(&otlpConfig.Redis, clientDBNames.TraceDBName, otlpConfig.Traces.SyncDuration, otlpConfig.Traces.BatchSize, traceRedisHandlerLogTag)
	nodeIP := os.Getenv("NODE_IP")

	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while creating redis client ", err)
	}

	handler := &TraceRedisHandler{
		redisHandler: redisHandler,
		ctx:          context.Background(),
		config:       otlpConfig,
		nodeIP:       nodeIP,
	}

	return handler, nil
}

func (h *TraceRedisHandler) CheckRedisConnection() error {
	return h.redisHandler.CheckRedisConnection()
}

func (h *TraceRedisHandler) PutTraceSource(traceId string, spanId string) error {
	return h.PutTraceData(traceId, spanId, h.nodeIP)
}

func (h *TraceRedisHandler) PutTraceData(traceId string, spanId string, spanJSON string) error {

	if err := h.redisHandler.CheckRedisConnection(); err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while checking redis conn ", err)
		return err
	}

	spanJsonMap := make(map[string]string)
	spanJsonMap[spanId] = spanJSON
	if err := h.redisHandler.HMSetPipeline(traceId, spanJsonMap, time.Duration(h.config.Traces.Ttl)*time.Second); err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while setting trace details for traceId %s: %v\n", traceId, err)
		return err
	}
	return nil
}

func (h *TraceRedisHandler) SyncPipeline() {
	h.redisHandler.SyncPipeline()
}
