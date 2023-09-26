package utils

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkconfig "github.com/zerok-ai/zk-utils-go/storage/redis/config"
	zktick "github.com/zerok-ai/zk-utils-go/ticker"
	"time"
)

var redisHandlerLogTag = "RedisHandler"

type RedisHandler struct {
	RedisClient  *redis.Client
	ctx          context.Context
	config       *zkconfig.RedisConfig
	dbName       string
	Pipeline     redis.Pipeliner
	ticker       *zktick.TickerTask
	count        int
	startTime    time.Time
	batchSize    int
	syncInterval int
}

func NewRedisHandler(redisConfig *zkconfig.RedisConfig, dbName string, syncInterval int, batchSize int) (*RedisHandler, error) {
	handler := &RedisHandler{
		ctx:    context.Background(),
		config: redisConfig,
		dbName: dbName,
	}

	err := handler.InitializeRedisConn()
	if err != nil {
		logger.Error(redisHandlerLogTag, "Error while initializing redis connection ", err)
		return nil, err
	}

	handler.Pipeline = handler.RedisClient.Pipeline()

	timerDuration := time.Duration(syncInterval) * time.Millisecond
	handler.ticker = zktick.GetNewTickerTask("sync_pipeline", timerDuration, handler.syncPipeline)
	handler.ticker.Start()

	handler.syncInterval = syncInterval
	handler.batchSize = batchSize

	return handler, nil
}

func (h *RedisHandler) InitializeRedisConn() error {
	db := h.config.DBs[h.dbName]
	redisAddr := h.config.Host + ":" + h.config.Port
	opt := &redis.Options{
		Addr:     redisAddr,
		Password: h.config.Password,
		DB:       db,
	}
	redisClient := redis.NewClient(opt)

	h.RedisClient = redisClient
	err := h.PingRedis()
	if err != nil {
		return err
	}
	return nil
}

func (h *RedisHandler) Set(key string, value interface{}) error {
	statusCmd := h.RedisClient.Set(h.ctx, key, value, 0)
	return statusCmd.Err()
}

func (h *RedisHandler) SetNX(key string, value interface{}) error {
	statusCmd := h.RedisClient.SetNX(h.ctx, key, value, 0)
	return statusCmd.Err()
}

func (h *RedisHandler) HSet(key string, value interface{}) error {
	statusCmd := h.RedisClient.HSet(h.ctx, key, value, 0)
	return statusCmd.Err()
}

func (h *RedisHandler) PingRedis() error {
	redisClient := h.RedisClient
	if redisClient == nil {
		logger.Error(redisHandlerLogTag, "Redis client is nil.")
		return fmt.Errorf("redis client is nil")
	}
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Error(redisHandlerLogTag, "Error caught while pinging redis ", err)
		return err
	}
	return nil
}

func (h *RedisHandler) CheckRedisConnection() error {
	err := h.PingRedis()
	if err != nil {
		//Closing redis connection.
		err := h.CloseConnection()
		if err != nil {
			logger.Error(redisHandlerLogTag, "Failed to close Redis connection: ", err)
			return err
		}
		err = h.InitializeRedisConn()
		if err != nil {
			logger.Error(redisHandlerLogTag, "Error while initializing redis connection ", err)
			return err
		}
	}
	return nil
}

func (h *RedisHandler) syncPipeline() {
	syncDuration := time.Duration(h.syncInterval) * time.Millisecond
	if h.count > h.batchSize || time.Since(h.startTime) >= syncDuration {
		_, err := h.Pipeline.Exec(h.ctx)
		if err != nil {
			logger.Error(traceRedisHandlerLogTag, "Error while syncing data to redis ", err)
			return
		}
		logger.Debug(traceRedisHandlerLogTag, "Pipeline synchronized on batchsize/syncDuration")

		h.count = 0
		h.startTime = time.Now()
	}
}

func (h *RedisHandler) CloseConnection() error {
	return h.RedisClient.Close()
}

func (h *RedisHandler) forceSync() {
	_, err := h.Pipeline.Exec(h.ctx)
	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while force syncing data to redis ", err)
		return
	}
}

func (h *RedisHandler) shutdown() {
	h.forceSync()
	err := h.CloseConnection()
	if err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while closing redis conn.")
		return
	}
}
