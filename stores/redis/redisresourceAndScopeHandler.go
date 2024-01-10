package redis

import (
	"encoding/json"
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/storage/redis/clientDBNames"
	"sync"
	"time"
)

type ResourceAndScopeAttributesHandler struct {
	redisHandler         *RedisHandler
	existingResourceData sync.Map
	otlpConfig           *config.OtlpConfig
}

func NewResourceAndScopeAttributesHandler(config *config.OtlpConfig) (*ResourceAndScopeAttributesHandler, error) {
	handler := ResourceAndScopeAttributesHandler{}
	redisHandler, err := NewRedisHandler(&config.Redis, clientDBNames.ResourceAndScopeAttrDBName, config.Resources.SyncDuration, config.Resources.BatchSize, resourceLogTag)
	if err != nil {
		logger.Error(resourceLogTag, "Error while creating resource redis handler:", err)
		return nil, err
	}

	handler.redisHandler = redisHandler
	handler.existingResourceData = sync.Map{}
	handler.otlpConfig = config
	return &handler, nil
}

func (h *ResourceAndScopeAttributesHandler) SyncResourceAndScopeAttrData(key string, attrMap map[string]interface{}) error {
	if attrMap == nil {
		return fmt.Errorf("attrMap is nil")
	}

	if len(attrMap) == 0 {
		return nil
	}

	attrStr, _ := json.Marshal(attrMap)

	err := h.redisHandler.CheckRedisConnection()
	if err != nil {
		logger.Error(resourceLogTag, "Error while checking redis conn ", err)
		return err
	}

	_, ok := h.existingResourceData.Load(key)
	expiry := time.Duration(h.otlpConfig.Resources.Ttl) * time.Second
	if !ok {
		err = h.redisHandler.Set(key, attrStr, expiry)
		if err != nil {
			logger.Error(resourceLogTag, "Error while setting resource or scope data: ", err)
			return err
		}
		h.existingResourceData.Store(key, attrStr)
	} else {
		err = h.redisHandler.setExpiry(key, expiry)
		if err != nil {
			logger.Error(resourceLogTag, "Error while setting resource or scope data expiry: ", err)
			return err
		}
	}

	return nil
}

func (h *ResourceAndScopeAttributesHandler) SyncPipeline() {
	h.redisHandler.SyncPipeline()
}
