package badger

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	__ "github.com/zerok-ai/zk-utils-go/proto/opentelemetry"
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

func (h *TraceBadgerHandler) PutTraceData(traceId string, spanId string, spanProto []byte) error {
	key := traceId + "-" + spanId
	if err := h.badgerHandler.Set(key, spanProto, time.Duration(h.config.Traces.Ttl)*time.Second); err != nil {
		logger.ErrorF(traceBadgerHandlerLogTag, "Error while setting trace details for traceId %s: %v", traceId, err)
		return err
	}

	return nil
}

func (h *TraceBadgerHandler) SyncPipeline() {
	h.badgerHandler.StartCompaction()
}

func (h *TraceBadgerHandler) GetBulkDataForPrefixList(prefixList []string) (map[string]*__.OtelEnrichedRawSpanForProto, error) {
	prefix, err := h.badgerHandler.BulkGetForPrefix(prefixList)
	if err != nil {
		logger.Error(traceBadgerHandlerLogTag, fmt.Sprintf("Error while fetching data from badger for given tracePrefixList: %v", prefixList), err)
		return nil, err
	}

	finalResp := make(map[string]*__.OtelEnrichedRawSpanForProto)
	for k, value := range prefix {
		var d __.OtelEnrichedRawSpanForProto
		err := proto.Unmarshal([]byte(value), &d)
		if err != nil {
			logger.Error(traceBadgerHandlerLogTag, fmt.Sprintf("Error while unmarshalling data from badger for given tracePrefixList: %v", prefixList), err)
			continue
		}
		finalResp[k] = &d
	}

	return finalResp, nil
}
