package redis

import (
	"encoding/json"
	"errors"
	"github.com/zerok-ai/zk-observer/config"
	"github.com/zerok-ai/zk-observer/model"
	"github.com/zerok-ai/zk-utils-go/common"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/storage/redis/clientDBNames"
	"sync"
)

var resourceLogTag = "ResourceRedisHandler"

type ResourceRedisHandler struct {
	redisHandler         *RedisHandler
	existingResourceData sync.Map
	otlpConfig           *config.OtlpConfig
}

func NewResourceDetailsHandler(config *config.OtlpConfig) (*ResourceRedisHandler, error) {
	handler := ResourceRedisHandler{}
	redisHandler, err := NewRedisHandler(&config.Redis, clientDBNames.PodDetailsDBName, config.Resources.SyncDuration, config.Resources.BatchSize, resourceLogTag)
	if err != nil {
		logger.Error(resourceLogTag, "Error while creating resource redis handler:", err)
		return nil, err
	}

	handler.redisHandler = redisHandler
	handler.existingResourceData = sync.Map{}
	handler.otlpConfig = config
	return &handler, nil
}

func (h *ResourceRedisHandler) SyncResourceData(resourceIp string, attrMap map[string]interface{}) error {
	if len(attrMap) == 0 {
		//Nothing to sync.
		return nil
	}

	err := h.redisHandler.CheckRedisConnection()
	if err != nil {
		logger.Error(resourceLogTag, "Error while checking redis conn ", err)
		return err
	}

	if common.IsEmpty(resourceIp) {
		logger.Debug(resourceLogTag, "Skipping saving resource data since resource Ip is empty")
		return errors.New("resourceIp is empty")
	}

	_, ok := h.existingResourceData.Load(resourceIp)
	if !ok {
		filteredResourceData := make(map[string]string)
		telemetryData := model.CreateTelemetryDetails(attrMap)
		var telemetryDataJSON []byte
		telemetryDataJSON, err = json.Marshal(telemetryData)
		if err != nil {
			logger.Debug(resourceLogTag, "Error encoding telemetryData for service %s: %v\n", telemetryData.ServiceName, err)
			return err
		}
		filteredResourceData["telemetry"] = string(telemetryDataJSON)
		err = h.redisHandler.HMSet(resourceIp, filteredResourceData)
		if err != nil {
			logger.Error(resourceLogTag, "Error while setting resource data: ", err)
			return err
		}
		h.existingResourceData.Store(resourceIp, filteredResourceData)
	}

	return nil
}

func (h *ResourceRedisHandler) SyncPipeline() {
	h.redisHandler.SyncPipeline()
}
