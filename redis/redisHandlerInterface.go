package redis

import (
	"time"
)

// RedisHandlerInterface defines the methods that the RedisHandler must implement.
type RedisHandlerInterface interface {
	InitializeRedisConn() error
	Set(key string, value interface{}) error
	Get(key string) (string, error)
	SetNX(key string, value interface{}) error
	HSet(key string, value interface{}) error
	HMSet(key string, value interface{}) error
	HMSetPipeline(key string, value map[string]string, expiration time.Duration) error
	GetKeysByPattern(pattern string) ([]string, error)
	SetWithTTL(key string, value interface{}, ttl time.Duration) error
	RemoveKey(key string) error
	RenameKeyWithTTL(oldKey string, newKey string, ttl time.Duration) error
	SetNXPipeline(key string, value interface{}, expiration time.Duration) error
	SAddPipeline(key string, value interface{}, expiration time.Duration) error
}
