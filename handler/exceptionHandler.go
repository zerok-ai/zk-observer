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

type ExceptionHandler struct {
	redisHandler          *utils.RedisHandler
	existingExceptionData sync.Map
	otlpConfig            *config.OtlpConfig
}

func NewExceptionHandler(config *config.OtlpConfig) (*ExceptionHandler, error) {
	handler := ExceptionHandler{}
	exceptionRedisHandler, err := utils.NewRedisHandler(&config.Redis, dbName)
	if err != nil {
		logger.Error(TRACE_LOG_TAG, "Error while creating exception redis handler:", err)
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
		hash = zkcommon.Generate256SHA(exception.Message, exception.Type, exception.Stacktrace)
		_, ok := th.existingExceptionData.Load(hash)
		if !ok {
			exceptionJSON, err := json.Marshal(exception)
			if err != nil {
				logger.Debug(TRACE_LOG_TAG, "Error encoding exception details for spanID %s: %v\n", spanId, err)
				return "", err
			}
			err = th.redisHandler.SetNX(hash, exceptionJSON)
			if err != nil {
				logger.Error(TRACE_LOG_TAG, "Error while saving exception to redis for span Id ", spanId, " with error ", err)
				return "", err
			}
			th.existingExceptionData.Store(hash, true)
		}
	} else {
		logger.Error(TRACE_LOG_TAG, "Could not find stacktrace for exception for span Id ", spanId)
		return "", fmt.Errorf("no stacktrace for the expcetion")
	}
	return hash, nil
}

func CreateExceptionDetails(event *tracev1.Span_Event) *model.ExceptionDetails {
	exceptionAttr := event.Attributes
	logger.Debug(TRACE_LOG_TAG, "Exception attributes ", exceptionAttr)
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
