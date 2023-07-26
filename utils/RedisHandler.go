package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"time"
)

var dbName = "traces"
var REDIS_LOG_TAG = "RedisHandler"

type RedisHandler struct {
	redisClient *redis.Client
	ctx         context.Context
	timer       *time.Timer
	count       int
	startTime   time.Time
	config      *config.RedisConfig
}

func NewRedisHandler(redisConfig *config.RedisConfig) (*RedisHandler, error) {
	handler := &RedisHandler{
		ctx:    context.Background(),
		config: redisConfig,
		//TODO: Verify the Timer Duration here.
		timer:     time.NewTimer(time.Duration(redisConfig.TimerDuration) * time.Second),
		startTime: time.Now(),
	}

	err := handler.initializeRedisConn()
	if err != nil {
		logger.Error(REDIS_LOG_TAG, "Error while initializing redis connection ", err)
		return nil, err
	}

	go handler.syncTask()

	return handler, nil
}

func (h *RedisHandler) initializeRedisConn() error {
	db := h.config.DBs[dbName]
	redisAddr := h.config.Host + ":" + h.config.Port

	opt := &redis.Options{
		Addr: redisAddr,
		DB:   db,
	}
	redisClient := redis.NewClient(opt)

	err := h.pingRedis()
	if err != nil {
		return err
	}
	h.redisClient = redisClient
	return nil
}

func (h *RedisHandler) pingRedis() error {
	if err := h.redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Error(REDIS_LOG_TAG, "Error caught while pinging redis ", err)
		return err
	}
	return nil
}

func (h *RedisHandler) syncTask() {
	logger.Debug(REDIS_LOG_TAG, "Syncing Trace Details.")
	for range h.timer.C {
		h.syncPipeline()
	}
}

func (h *RedisHandler) syncPipeline() {
	//TODO: Verify the Sync Duration here.
	syncDuration := time.Duration(h.config.SyncDuration) * time.Second
	if h.count > h.config.BatchSize || time.Since(h.startTime) >= syncDuration {
		_, err := h.redisClient.Pipeline().Exec(h.ctx)
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
	err := h.pingRedis()
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
		spanDetails := value.(*model.SpanDetails)
		spanJSON, err := json.Marshal(spanDetails)
		if err != nil {
			fmt.Printf("Error encoding SpanDetails for spanID %s: %v\n", spanIdStr, err)
			return true
		}
		spanJsonMap[spanIdStr] = string(spanJSON)
		return true
	})

	pipeline := h.redisClient.Pipeline()
	ctx := context.Background()
	pipeline.HMSet(ctx, traceID, spanJsonMap)
	//TODO: Verify the unit of ttl here.
	pipeline.Expire(ctx, traceID, time.Duration(h.config.Ttl)*time.Second)
	return nil
}

func (h *RedisHandler) forceSync() {
	_, err := h.redisClient.Pipeline().Exec(h.ctx)
	if err != nil {
		logger.Error(REDIS_LOG_TAG, "Error while force syncing data to redis ", err)
		return
	}
}

func (h *RedisHandler) shutdown() {
	h.forceSync()
}
