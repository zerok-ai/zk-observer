package utils

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zktick "github.com/zerok-ai/zk-utils-go/ticker"
	"time"
)

var traceDbName = "traces"
var traceRedisHandlerLogTag = "TraceRedisHandler"

type TraceRedisHandler struct {
	redisHandler *RedisHandler
	ctx          context.Context
	ticker       *zktick.TickerTask
	count        int
	startTime    time.Time
	config       *config.OtlpConfig
	pipeline     redis.Pipeliner
}

func NewTracesRedisHandler(otlpConfig *config.OtlpConfig) (*TraceRedisHandler, error) {
	redisHandler, err := NewRedisHandler(&otlpConfig.Redis, traceDbName)

	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while creating redis client ", err)
	}

	handler := &TraceRedisHandler{
		redisHandler: redisHandler,
		ctx:          context.Background(),
		config:       otlpConfig,
		startTime:    time.Now(),
	}

	handler.pipeline = handler.redisHandler.redisClient.Pipeline()

	timerDuration := time.Duration(otlpConfig.Traces.TimerDuration) * time.Millisecond
	handler.ticker = zktick.GetNewTickerTask("sync_pipeline", timerDuration, handler.syncPipeline)
	handler.ticker.Start()

	return handler, nil
}

func (h *TraceRedisHandler) syncPipeline() {
	syncDuration := time.Duration(h.config.Traces.SyncDuration) * time.Millisecond
	if h.count > h.config.Traces.BatchSize || time.Since(h.startTime) >= syncDuration {
		_, err := h.pipeline.Exec(h.ctx)
		if err != nil {
			logger.Error(traceRedisHandlerLogTag, "Error while syncing data to redis ", err)
			return
		}
		logger.Debug(traceRedisHandlerLogTag, "Pipeline synchronized on batchsize/syncDuration")

		h.count = 0
		h.startTime = time.Now()
	}
}

func (h *TraceRedisHandler) PutTraceData(traceID string, traceDetails *model.TraceDetails) error {
	err := h.redisHandler.CheckRedisConnection()
	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while checking redis conn ", err)
	}
	spanJsonMap := make(map[string]string)
	traceDetails.SpanDetailsMap.Range(func(spanId, value interface{}) bool {
		spanIdStr := spanId.(string)
		spanDetails := value.(model.SpanDetails)
		spanJSON, err := json.Marshal(spanDetails)
		if err != nil {
			logger.Debug(traceRedisHandlerLogTag, "Error encoding SpanDetails for spanID %s: %v\n", spanIdStr, err)
			return true
		}
		spanJsonMap[spanIdStr] = string(spanJSON)
		return true
	})

	ctx := context.Background()
	logger.Debug(traceRedisHandlerLogTag, "Len of redis pipeline ", h.pipeline.Len())
	h.pipeline.HMSet(ctx, traceID, spanJsonMap)
	logger.Debug(traceRedisHandlerLogTag, "Len of redis pipeline ", h.pipeline.Len())
	h.pipeline.Expire(ctx, traceID, time.Duration(h.config.Traces.Ttl)*time.Second)
	h.count++
	h.syncPipeline()
	return nil
}

func (h *TraceRedisHandler) forceSync() {
	_, err := h.pipeline.Exec(h.ctx)
	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while force syncing data to redis ", err)
		return
	}
}

func (h *TraceRedisHandler) shutdown() {
	h.forceSync()
	err := h.redisHandler.CloseConnection()
	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while closing redis conn.")
		return
	}
}
