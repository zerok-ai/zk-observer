package badger

import (
	"context"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/storage/badger"
	"time"
)

var traceBadgerHandlerLogTag = "TraceBadgerHandler"

type TraceBadgerHandler struct {
	badgerHandler *badger.BadgerStoreHandler
	ctx           context.Context
	config        *config.OtlpConfig
}

func NewTracesBadgerHandler(otlpConfig *config.OtlpConfig) (*TraceBadgerHandler, error) {
	badgerHandler, err := badger.NewBadgerHandler(&otlpConfig.Badger)

	if err != nil {
		logger.Error(traceBadgerHandlerLogTag, "Error while creating badger client ", err)
	}

	handler := &TraceBadgerHandler{
		badgerHandler: badgerHandler,
		ctx:           context.Background(),
		config:        otlpConfig,
	}

	return handler, nil
}

func (h *TraceBadgerHandler) PutTraceData(traceId string, spanId string, spanJSON string) error {
	key := traceId + "-" + spanId
	logger.InfoF(traceBadgerHandlerLogTag, "Setting %s as %s", key, spanJSON)
	if err := h.badgerHandler.Set(key, spanJSON, int64(time.Duration(h.config.Traces.Ttl)*time.Second)); err != nil {
		logger.ErrorF(traceBadgerHandlerLogTag, "Error while setting trace details for traceId %s: %v", traceId, err)
		return err
	}
	logger.InfoF(traceBadgerHandlerLogTag, "Value at %s successfully set.", key)
	return nil
}

func (h *TraceBadgerHandler) SyncPipeline() {
	h.badgerHandler.StartCompaction()
}
