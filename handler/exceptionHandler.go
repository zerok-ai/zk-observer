package handler

import (
	"encoding/json"
	"fmt"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/model"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	zkcommon "github.com/zerok-ai/zk-utils-go/common"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"sync"
)

var dbName = "exception"
var exceptionLogTag = "ExceptionHandler"

type ExceptionHandler struct {
	redisHandler          *utils.RedisHandler
	existingExceptionData sync.Map
	otlpConfig            *config.OtlpConfig
}

func NewExceptionHandler(config *config.OtlpConfig) (*ExceptionHandler, error) {
	handler := ExceptionHandler{}
	exceptionRedisHandler, err := utils.NewRedisHandler(&config.Redis, dbName, config.Exception.SyncDuration, config.Exception.BatchSize, exceptionLogTag)
	if err != nil {
		logger.Error(exceptionLogTag, "Error while creating exception redis handler:", err)
		return nil, err
	}

	handler.redisHandler = exceptionRedisHandler
	handler.existingExceptionData = sync.Map{}
	handler.otlpConfig = config
	return &handler, nil
}

func (th *ExceptionHandler) SyncExceptionData(exception *model.ExceptionDetails, spanId string) (string, error) {
	hash := ""
	if len(exception.Stacktrace) > 0 {
		err := th.redisHandler.CheckRedisConnection()
		if err != nil {
			logger.Error(exceptionLogTag, "Error while checking redis conn ", err)
			return "", err
		}
		hash = zkcommon.Generate256SHA(exception.Message, exception.Type, exception.Stacktrace)
		_, ok := th.existingExceptionData.Load(hash)
		if !ok {
			exceptionJSON, err := json.Marshal(exception)
			if err != nil {
				logger.Error(exceptionLogTag, "Error encoding exception details for spanID %s: %v\n", spanId, err)
				return "", err
			}
			//Directly setting this to redis, because each resource will be only be written once. So no need to create a pipeline.
			err = th.redisHandler.SetNXPipeline(hash, exceptionJSON, 0)
			if err != nil {
				logger.Error(exceptionLogTag, "Error while setting exception details for spanID %s: %v\n", spanId, err)
				return "", err
			}
			th.existingExceptionData.Store(hash, true)
		}
	} else {
		logger.Error(exceptionLogTag, "Could not find stacktrace for exception for span Id ", spanId)
		return "", fmt.Errorf("no stacktrace for the expcetion")
	}
	return hash, nil
}

func CreateExceptionDetails(event *tracev1.Span_Event) *model.ExceptionDetails {
	exceptionAttr := event.Attributes
	//logger.Debug(exceptionLogTag, "Exception attributes ", exceptionAttr)
	exception := model.ExceptionDetails{}
	for _, attr := range exceptionAttr {
		switch attr.Key {
		case "exception.stacktrace":
			exception.Stacktrace = attr.Value.GetStringValue()
		case "exception.message":
			exception.Message = attr.Value.GetStringValue()
		case "exception.type":
			exception.Type = attr.Value.GetStringValue()
		}
	}
	return &exception
}
