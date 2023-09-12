package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zktick "github.com/zerok-ai/zk-utils-go/ticker"
	"time"
)

var redisHandlerTag = "RedisHandler"

var dbName = "traces"
var REDIS_LOG_TAG = "RedisHandler"

type RedisHandler struct {
	redisClient *redis.Client
	ctx         context.Context
	ticker      *zktick.TickerTask
	count       int
	startTime   time.Time
	config      *config.RedisConfig
	pipeline    redis.Pipeliner
}

func NewRedisHandler(redisConfig *config.RedisConfig) (*RedisHandler, error) {
	handler := &RedisHandler{
		ctx:       context.Background(),
		config:    redisConfig,
		startTime: time.Now(),
	}

	err := handler.initializeRedisConn()
	if err != nil {
		logger.Error(REDIS_LOG_TAG, "Error while initializing redis connection ", err)
		return nil, err
	}
	handler.pipeline = handler.redisClient.Pipeline()

	timerDuration := time.Duration(redisConfig.TimerDuration) * time.Millisecond
	handler.ticker = zktick.GetNewTickerTask("sync_pipeline", timerDuration, handler.syncPipeline)
	handler.ticker.Start()

	return handler, nil
}

func (h *RedisHandler) initializeRedisConn() error {
	db := h.config.DBs[dbName]
	redisAddr := h.config.Host + ":" + h.config.Port
	logger.Debug(redisHandlerTag, "Redis Address ", redisAddr)
	logger.Debug(redisHandlerTag, "Redis Password ", h.config.Password)
	opt := &redis.Options{
		Addr:     redisAddr,
		Password: h.config.Password,
		DB:       db,
	}
	redisClient := redis.NewClient(opt)

	//redisClient.Set(h.ctx, "Test", "Testing", 0)

	err := h.pingRedis(redisClient)
	if err != nil {
		return err
	}
	h.redisClient = redisClient
	return nil
}

func (h *RedisHandler) pingRedis(redisClient *redis.Client) error {
	if redisClient == nil {
		logger.Error(REDIS_LOG_TAG, "Redis client is nil.")
		return fmt.Errorf("redis client is nil")
	}
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Error(REDIS_LOG_TAG, "Error caught while pinging redis ", err)
		return err
	}
	return nil
}

func (h *RedisHandler) syncPipeline() {
	syncDuration := time.Duration(h.config.SyncDuration) * time.Millisecond
	if h.count > h.config.BatchSize || time.Since(h.startTime) >= syncDuration {
		_, err := h.pipeline.Exec(h.ctx)
		if err != nil {
			logger.Error(REDIS_LOG_TAG, "Error while syncing data to redis ", err)
			return
		}
		logger.Debug(REDIS_LOG_TAG, "Pipeline synchronized on batchsize/syncDuration")

		h.count = 0
		h.startTime = time.Now()
	}
}

func (h *RedisHandler) PutTraceData(traceID string, traceDetails *model.TraceDetails) error {
	err := h.pingRedis(h.redisClient)
	if err != nil {
		//Closing redis connection.
		err := h.redisClient.Close()
		if err != nil {
			logger.Error(REDIS_LOG_TAG, "Failed to close Redis connection: ", err)
			return err
		}
		err = h.initializeRedisConn()
		if err != nil {
			//TODO: Confirm with shivam if we should return here or not.
			logger.Error(REDIS_LOG_TAG, "Error while initializing redis connection ", err)
			return err
		}
	}

	spanJsonMap := make(map[string]string)
	traceDetails.SpanDetailsMap.Range(func(spanId, value interface{}) bool {
		spanIdStr := spanId.(string)
		spanDetails := value.(model.SpanDetails)
		spanJSON, err := json.Marshal(spanDetails)
		if err != nil {
			logger.Debug(REDIS_LOG_TAG, "Error encoding SpanDetails for spanID %s: %v\n", spanIdStr, err)
			return true
		}
		spanJsonMap[spanIdStr] = string(spanJSON)
		return true
	})

	ctx := context.Background()
	logger.Debug(REDIS_LOG_TAG, "Len of redis pipeline ", h.pipeline.Len())
	h.pipeline.HMSet(ctx, traceID, spanJsonMap)
	logger.Debug(REDIS_LOG_TAG, "Len of redis pipeline ", h.pipeline.Len())
	h.pipeline.Expire(ctx, traceID, time.Duration(h.config.Ttl)*time.Second)
	h.count++
	h.syncPipeline()
	return nil
}

func (h *RedisHandler) forceSync() {
	_, err := h.pipeline.Exec(h.ctx)
	if err != nil {
		logger.Error(REDIS_LOG_TAG, "Error while force syncing data to redis ", err)
		return
	}
}

func (h *RedisHandler) shutdown() {
	h.forceSync()
}
