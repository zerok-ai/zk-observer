package redis

import (
	"context"
	"github.com/zerok-ai/zk-observer/common"
	"github.com/zerok-ai/zk-observer/config"
	"github.com/zerok-ai/zk-utils-go/ds"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/storage/redis/clientDBNames"
)

var ServiceListRedisHandlerLogTag = "ServiceListRedisHandler"

type ServiceListRedisHandler struct {
	redisHandler *RedisHandler
	ctx          context.Context
	config       *config.OtlpConfig
	serviceList  ds.Set[string]
}

func NewServiceListRedisHandler(otlpConfig *config.OtlpConfig) (*ServiceListRedisHandler, error) {
	redisHandler, err := NewRedisHandler(&otlpConfig.Redis, clientDBNames.ServiceListDBName, otlpConfig.Services.SyncDuration, otlpConfig.Services.BatchSize, ServiceListRedisHandlerLogTag)
	if err != nil {
		logger.Error(ServiceListRedisHandlerLogTag, "Error while creating redis client ", err)
	}

	handler := &ServiceListRedisHandler{
		redisHandler: redisHandler,
		ctx:          context.Background(),
		config:       otlpConfig,
		serviceList:  make(ds.Set[string]),
	}

	handler.serviceList.Add(common.ScenarioWorkloadGenericServiceNameKey)
	_ = handler.PutServiceListData(common.ServiceListKey, common.ScenarioWorkloadGenericServiceNameKey)
	return handler, nil
}

func (h *ServiceListRedisHandler) CheckRedisConnection() error {
	return h.redisHandler.CheckRedisConnection()
}

func (h *ServiceListRedisHandler) PutServiceListData(key string, serviceName string) error {
	if err := h.redisHandler.CheckRedisConnection(); err != nil {
		logger.Error(traceRedisHandlerLogTag, "Error while checking redis conn ", err)
		return err
	}

	h.serviceList.Add(serviceName)

	if err := h.redisHandler.HSet(key, h.serviceList.GetAll()); err != nil {
		logger.Error(ServiceListRedisHandlerLogTag, "Error while setting service name: %s, %v", serviceName, err)
		return err
	}

	return nil
}

func (h *ServiceListRedisHandler) SyncPipeline() {
	h.redisHandler.SyncPipeline()
}
