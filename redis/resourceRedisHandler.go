package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"strings"
	"sync"
)

var resourceDbName = "resource"
var resourceLogTag = "ResourceRedisHandler"
var skipResourceDataError = fmt.Errorf("skipping saving resource data")

type ResourceRedisHandler struct {
	redisHandler         *RedisHandler
	existingResourceData sync.Map
	otlpConfig           *config.OtlpConfig
}

func NewResourceDetailsHandler(config *config.OtlpConfig) (*ResourceRedisHandler, error) {
	handler := ResourceRedisHandler{}
	redisHandler, err := NewRedisHandler(&config.Redis, resourceDbName, config.Resources.SyncDuration, config.Resources.BatchSize, resourceLogTag)
	if err != nil {
		logger.Error(resourceLogTag, "Error while creating resource redis handler:", err)
		return nil, err
	}

	handler.redisHandler = redisHandler
	handler.existingResourceData = sync.Map{}
	handler.otlpConfig = config
	return &handler, nil
}

func (h *ResourceRedisHandler) SyncResourceData(spanDetailsInput *map[string]interface{}, attrMap map[string]interface{}) error {
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
	resourceIp, err := h.getResourceIP(spanDetails)
	if err != nil {
		if errors.Is(err, skipResourceDataError) {
			return nil
		}
		logger.Error(resourceLogTag, "Error while getting resource IP ", err)
		return err
	}

	_, ok := h.existingResourceData.Load(resourceIp)
	if !ok {
		filters := []string{"service", "telemetry"}
		filteredResourceData := h.filterResourceData(filters, attrMap)
		err := h.redisHandler.HMSetPipeline(resourceIp, filteredResourceData, 0)
		if err != nil {
			logger.Error(resourceLogTag, "Error while setting resource data: ", err)
			return err
		}
		h.existingResourceData.Store(resourceIp, filteredResourceData)
	}

	return nil
}

func (h *ResourceRedisHandler) getResourceIP(spanDetails map[string]interface{}) (string, error) {
	resourceIp := ""
	spanKindStr, ok := spanDetails[common.SpanKindKey].(model.SpanKind)
	if ok && spanKindStr == model.SpanKindClient {
		sourceIp := spanDetails[common.SourceIpKey]
		if sourceIp != nil {
			sourceIpStr, ok := sourceIp.(string)
			if ok {
				resourceIp = sourceIpStr
			}
		}
	} else if ok && spanKindStr == model.SpanKindServer {
		destIp := spanDetails[common.DestIpKey]
		if destIp != nil {
			destIpStr, ok := destIp.(string)
			if ok {
				resourceIp = destIpStr
			}
		}
	} else {
		//No need to save resource details.
		logger.Debug(resourceLogTag, "Skipping saving resource data for spanKind ", spanKindStr)
		return "", skipResourceDataError
	}
	if len(resourceIp) == 0 {
		logger.Debug(resourceLogTag, "Skipping saving resource data since resource Ip is empty for spanKind ", spanKindStr)
		return "", fmt.Errorf("resourceIp is empty")
	}
	return resourceIp, nil
}

func (h *ResourceRedisHandler) filterResourceData(filters []string, attrMap map[string]interface{}) map[string]string {
	finalMap := make(map[string]string)

	for _, filter := range filters {
		tempMap := map[string]interface{}{}
		count := 0
		for key, value := range attrMap {
			if strings.HasPrefix(key, filter) {
				tempMap[key] = value
				count++
			}
		}
		if count > 0 {
			tempMapJSON, err := json.Marshal(tempMap)
			if err != nil {
				logger.Error(resourceLogTag, "Error encoding resource details: %v\n", err)
				continue
			}
			finalMap[filter] = string(tempMapJSON)
		}
	}
	return finalMap
}

func (h *ResourceRedisHandler) SyncPipeline() {
	h.redisHandler.SyncPipeline()
}
