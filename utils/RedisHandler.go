package utils

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
)

var redisHandlerTag = "RedisHandler"

var REDIS_LOG_TAG = "RedisHandler"

type RedisHandler struct {
	redisClient *redis.Client
	ctx         context.Context
	config      *config.RedisConfig
	dbName      string
}

func NewRedisHandler(redisConfig *config.RedisConfig, dbName string) (*RedisHandler, error) {
	handler := &RedisHandler{
		ctx:    context.Background(),
		config: redisConfig,
		dbName: dbName,
	}

	err := handler.InitializeRedisConn()
	if err != nil {
		logger.Error(REDIS_LOG_TAG, "Error while initializing redis connection ", err)
		return nil, err
	}

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

	h.redisClient = redisClient
	err := h.PingRedis()
	if err != nil {
		return err
	}
	return nil
}

func (h *RedisHandler) SetValue(key string, value interface{}) error {
	statusCmd := h.redisClient.Set(h.ctx, key, value, 0)
	return statusCmd.Err()
}

func (h *RedisHandler) PingRedis() error {
	redisClient := h.redisClient
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

func (h *RedisHandler) CheckRedisConnection() error {
	err := h.PingRedis()
	if err != nil {
		//Closing redis connection.
		err := h.CloseConnection()
		if err != nil {
			logger.Error(TRACES_REDIS_LOG_TAG, "Failed to close Redis connection: ", err)
			return err
		}
		err = h.InitializeRedisConn()
		if err != nil {
			logger.Error(TRACES_REDIS_LOG_TAG, "Error while initializing redis connection ", err)
			return err
		}
	}
	return nil
}

func (h *RedisHandler) CloseConnection() error {
	return h.redisClient.Close()
}
