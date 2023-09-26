package handler

import (
	"encoding/json"
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"strings"
	"sync"
)

var resourceDbName = "resource"
var resourceLogTag = "ResourceDetailsHandler"

type ResourceDetailsHandler struct {
	redisHandler         *utils.RedisHandler
	existingResourceData sync.Map
	otlpConfig           *config.OtlpConfig
}

func NewResourceDetailsHandler(config *config.OtlpConfig) (*ResourceDetailsHandler, error) {
	handler := ResourceDetailsHandler{}
	redisHandler, err := utils.NewRedisHandler(&config.Redis, resourceDbName, config.Resources.SyncDuration, config.Resources.BatchSize, resourceLogTag)
	if err != nil {
		logger.Error(resourceLogTag, "Error while creating resource redis handler:", err)
		return nil, err
	}

	handler.redisHandler = redisHandler
	handler.existingResourceData = sync.Map{}
	handler.otlpConfig = config
	return &handler, nil
}

func (th *ResourceDetailsHandler) SyncResourceData(spanDetailsInput *map[string]interface{}, attrMap map[string]interface{}) error {
	if spanDetailsInput == nil {
		return fmt.Errorf("spanDetails are nil")
	}
	err := th.redisHandler.CheckRedisConnection()
	if err != nil {
		logger.Error(resourceLogTag, "Error while checking redis conn ", err)
		return err
	}
	spanDetails := *spanDetailsInput
	//logger.Debug(spanFilteringLogTag, "Span details are: ", spanDetails)
	if len(attrMap) > 0 {
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
			return nil
		}
		if len(resourceIp) == 0 {
			logger.Debug(resourceLogTag, "Skipping saving resource data since resource Ip is empty for spanKind ", spanKindStr)
			return nil
		}
		existingValue, ok := th.existingResourceData.Load(resourceIp)
		if !ok {
			filters := []string{"service", "telemetry"}
			filteredResourceData := th.FilterResourceData(filters, attrMap)
			logger.Debug(resourceLogTag, "Resource data is ", filteredResourceData)
			//Directly setting this to redis, because each resource will be only be written once. So no need to create a pipeline.
			err := th.redisHandler.HMSetPipeline(resourceIp, filteredResourceData, 0)
			if err != nil {
				logger.Error(resourceLogTag, "Error while setting resource data: ", err)
				return err
			}
			th.existingResourceData.Store(resourceIp, filteredResourceData)
		}
		logger.Debug(resourceLogTag, "Existing resource attrMap is ", existingValue)
	}
	return nil
}

func (th *ResourceDetailsHandler) FilterResourceData(filters []string, attrMap map[string]interface{}) map[string]interface{} {
	finalMap := map[string]interface{}{}

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
