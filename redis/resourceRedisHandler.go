package redis

import (
	"encoding/json"
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"sync"
)

var resourceLogTag = "ResourceRedisHandler"
var skipResourceDataError = fmt.Errorf("skipping saving resource data")

type ResourceRedisHandler struct {
	redisHandler         *RedisHandler
	existingResourceData sync.Map
	otlpConfig           *config.OtlpConfig
}

func NewResourceDetailsHandler(config *config.OtlpConfig) (*ResourceRedisHandler, error) {
	handler := ResourceRedisHandler{}
	redisHandler, err := NewRedisHandler(&config.Redis, common.PodDetailsDbName, config.Resources.SyncDuration, config.Resources.BatchSize, resourceLogTag)
	if err != nil {
		logger.Error(resourceLogTag, "Error while creating resource redis handler:", err)
		return nil, err
	}

	handler.redisHandler = redisHandler
	handler.existingResourceData = sync.Map{}
	handler.otlpConfig = config
	return &handler, nil
}

func (h *ResourceRedisHandler) SyncResourceData(spanDetailsInput *model.OTelSpanDetails, attrMap map[string]interface{}) error {
	if spanDetailsInput == nil {
		return fmt.Errorf("spanDetails are nil")
	}

	if len(attrMap) == 0 {
		//Nothing to sync.
		return nil
	}

	err := h.redisHandler.CheckRedisConnection()
	if err != nil {
		logger.Error(resourceLogTag, "Error while checking redis conn ", err)
		return err
	}

	spanDetails := *spanDetailsInput
	resourceIp := spanDetails.GetResourceIP()
	if len(resourceIp) == 0 {
		logger.Debug(resourceLogTag, "Skipping saving resource data since resource Ip is empty for spanKind ", spanDetails.SpanKind)
	}

	_, ok := h.existingResourceData.Load(resourceIp)
	if !ok {
		filteredResourceData := make(map[string]string)
		telemetryData := model.CreateTelemetryDetails(attrMap)
		telemetryDataJSON, err := json.Marshal(telemetryData)
		if err != nil {
			logger.Debug(resourceLogTag, "Error encoding telemetryData for service %s: %v\n", telemetryData.ServiceName, err)
			return err
		}
		filteredResourceData["telemetry"] = string(telemetryDataJSON)
		logger.Debug("Setting resource data ", filteredResourceData)
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
