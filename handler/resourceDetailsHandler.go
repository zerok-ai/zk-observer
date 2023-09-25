package handler

import (
	"encoding/json"
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/common"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
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
	redisHandler, err := utils.NewRedisHandler(&config.Redis, resourceDbName)
	if err != nil {
		logger.Error(resourceLogTag, "Error while creating resource redis handler:", err)
		return nil, err
	}

	handler.redisHandler = redisHandler
	handler.existingResourceData = sync.Map{}
	handler.otlpConfig = config
	return &handler, nil
}

func (th *ResourceDetailsHandler) SyncResourceData(spanId string, spanDetails map[string]interface{}, attrMap map[string]interface{}) error {
	if spanDetails == nil {
		return fmt.Errorf("spanDetails are nil")
	}
	if len(attrMap) > 0 {
		resourceIp := ""
		spanKindStr, ok := spanDetails[common.SpanKindKey].(string)
		if ok && spanKindStr == string(model.SpanKindClient) {
			resourceIp = spanDetails[common.SourceIpKey].(string)
		} else if ok && spanKindStr == string(model.SpanKindServer) {
			resourceIp = spanDetails[common.DestIpKey].(string)
		} else {
			//No need to save resource details.
			logger.Debug(resourceLogTag, "Skipping saving resource data for spanKind ", spanKindStr)
			return nil
		}
		if len(resourceIp) == 0 {
			logger.Debug(resourceLogTag, "Skipping saving resource data since resource Ip is empty for spanKind ", spanKindStr)
			return nil
		}
		_, ok = th.existingResourceData.Load(resourceIp)
		if !ok {
			resourceAttrJSON, err := json.Marshal(attrMap)
			if err != nil {
				logger.Error(resourceLogTag, "Error encoding resource details for spanID %s: %v\n", spanId, err)
				return err
			}
			err = th.redisHandler.SetNX(resourceIp, resourceAttrJSON)
			if err != nil {
				logger.Error(resourceLogTag, "Error while saving resource to redis for span Id ", spanId, " with error ", err)
				return err
			}
			th.existingResourceData.Store(resourceIp, true)
		}
	}
	return nil
}
