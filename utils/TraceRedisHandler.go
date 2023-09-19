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
var TRACES_REDIS_LOG_TAG = "TraceRedisHandler"

type TraceRedisHandler struct {
	redisHandler *RedisHandler
	ctx          context.Context
	ticker       *zktick.TickerTask
	count        int
	startTime    time.Time
	config       *config.RedisConfig
	pipeline     redis.Pipeliner
}

func NewTracesRedisHandler(redisConfig *config.RedisConfig) (*TraceRedisHandler, error) {
	redisHandler, err := NewRedisHandler(redisConfig, traceDbName)

	if err != nil {
		logger.Error(TRACES_REDIS_LOG_TAG, "Error while creating redis client ", err)
	}

	handler := &TraceRedisHandler{
		redisHandler: redisHandler,
		ctx:          context.Background(),
		config:       redisConfig,
		startTime:    time.Now(),
	}

	handler.pipeline = handler.redisHandler.redisClient.Pipeline()

	timerDuration := time.Duration(redisConfig.TimerDuration) * time.Millisecond
	handler.ticker = zktick.GetNewTickerTask("sync_pipeline", timerDuration, handler.syncPipeline)
	handler.ticker.Start()

	return handler, nil
}

func (h *TraceRedisHandler) syncPipeline() {
	syncDuration := time.Duration(h.config.SyncDuration) * time.Millisecond
	if h.count > h.config.BatchSize || time.Since(h.startTime) >= syncDuration {
		_, err := h.pipeline.Exec(h.ctx)
		if err != nil {
			logger.Error(TRACES_REDIS_LOG_TAG, "Error while syncing data to redis ", err)
			return
		}
		logger.Debug(TRACES_REDIS_LOG_TAG, "Pipeline synchronized on batchsize/syncDuration")

		h.count = 0
		h.startTime = time.Now()
	}
}

func (h *TraceRedisHandler) PutTraceData(traceID string, traceDetails *model.TraceDetails) error {
	err := h.redisHandler.CheckRedisConnection()
	if err != nil {
		logger.Error(TRACES_REDIS_LOG_TAG, "Error while checking redis conn ", err)
	}
	spanJsonMap := make(map[string]string)
	traceDetails.SpanDetailsMap.Range(func(spanId, value interface{}) bool {
		spanIdStr := spanId.(string)
		spanDetails := value.(model.SpanDetails)
		spanJSON, err := json.Marshal(spanDetails)
		if err != nil {
			logger.Debug(TRACES_REDIS_LOG_TAG, "Error encoding SpanDetails for spanID %s: %v\n", spanIdStr, err)
			return true
		}
		spanJsonMap[spanIdStr] = string(spanJSON)
		return true
	})

	ctx := context.Background()
	logger.Debug(TRACES_REDIS_LOG_TAG, "Len of redis pipeline ", h.pipeline.Len())
	h.pipeline.HMSet(ctx, traceID, spanJsonMap)
	logger.Debug(TRACES_REDIS_LOG_TAG, "Len of redis pipeline ", h.pipeline.Len())
	h.pipeline.Expire(ctx, traceID, time.Duration(h.config.Ttl)*time.Second)
	h.count++
	h.syncPipeline()
	return nil
}

func (h *TraceRedisHandler) forceSync() {
	_, err := h.pipeline.Exec(h.ctx)
	if err != nil {
		logger.Error(TRACES_REDIS_LOG_TAG, "Error while force syncing data to redis ", err)
		return
	}
}

func (h *TraceRedisHandler) shutdown() {
	h.forceSync()
	err := h.redisHandler.CloseConnection()
	if err != nil {
		logger.Error(TRACES_REDIS_LOG_TAG, "Error while closing redis conn.")
		return
	}
}
