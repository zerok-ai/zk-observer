package badger

import (
	"context"
	"fmt"
	"github.com/zerok-ai/zk-observer/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/storage/badger"
	"time"
)

// TODO: This should be renamed to badger handler.
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

func (h *TraceBadgerHandler) PutTraceData(traceId string, spanId string, spanProto []byte) error {
	key := traceId + "-o-" + spanId
	return h.PutData(key, spanProto)
}

// PutEbpfData Method to set ebpf data in badger
func (h *TraceBadgerHandler) PutEbpfData(traceId string, spanId string, ebpfData []byte) error {
	key := traceId + "-e-" + spanId
	return h.PutData(key, ebpfData)
}

func (h *TraceBadgerHandler) PutData(key string, data []byte) error {
	logger.InfoF(traceBadgerHandlerLogTag, "Setting %s as %s", key, data)
	if err := h.badgerHandler.Set(key, data, time.Duration(h.config.Traces.Ttl)*time.Second); err != nil {
		logger.ErrorF(traceBadgerHandlerLogTag, "Error while setting trace details for key with error ", key, err)
		return err
	}
	return nil
}

func (h *TraceBadgerHandler) SyncPipeline() {
	h.badgerHandler.StartCompaction()
}

func (h *TraceBadgerHandler) GetBulkDataForPrefixList(prefixList []string) (*map[string]string, error) {
	badgerDataMap, err := h.badgerHandler.BulkGetForPrefix(prefixList)
	if err != nil {
		logger.Error(traceBadgerHandlerLogTag, fmt.Sprintf("Error while fetching data from badger for given tracePrefixList: %v", prefixList), err)
		return nil, err
	}
	return &badgerDataMap, nil
}

// GetData Function to get badger data for a key
func (h *TraceBadgerHandler) GetData(key string) (string, error) {
	logger.Info(traceBadgerHandlerLogTag, fmt.Sprintf("Fetching data form badger for given key: %s", key))
	value, err := h.badgerHandler.Get(key)
	if err != nil {
		logger.Error(traceBadgerHandlerLogTag, fmt.Sprintf("Error while fetching data from badger for given key: %s", key), err)
		return "", err
	}
	logger.Info(traceBadgerHandlerLogTag, fmt.Sprintf("Fetched data form badger for given key: %s", key))
	return value, nil
}

func (h *TraceBadgerHandler) PrefixGet(prefix string) (map[string]string, error) {
	logger.Info(traceBadgerHandlerLogTag, fmt.Sprintf("Fetching data form badger for given prefix: %s", prefix))
	prefixMap, err := h.badgerHandler.BulkGetForPrefix([]string{prefix})
	if err != nil {
		logger.Error(traceBadgerHandlerLogTag, fmt.Sprintf("Error while fetching data from badger for given prefix: %s", prefix), err)
		return nil, err
	}
	logger.Info(traceBadgerHandlerLogTag, fmt.Sprintf("Fetched data form badger for given prefix: %s", prefix))
	return prefixMap, nil
}
