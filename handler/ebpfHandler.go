package handler

import (
	"github.com/golang/protobuf/proto"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/stores/badger"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkUtilsOtel "github.com/zerok-ai/zk-utils-go/proto"
	"github.com/zerok-ai/zk-utils-go/socket"
)

var ebpfHandlerLogTag = "ebpfHandler"

type EbpfHandler struct {
	tcpServer          socket.TCPServer
	traceBadgerHandler *badger.TraceBadgerHandler
}

func CreateAndStartEbpfHandler(config *config.OtlpConfig, traceBadgerHandler *badger.TraceBadgerHandler) *EbpfHandler {
	handler := EbpfHandler{}
	tcpServer := socket.CreateTCPServer(config.TcpServerConfig, handler.HandleData)
	handler.tcpServer = *tcpServer
	handler.traceBadgerHandler = traceBadgerHandler
	tcpServer.Start()
	return &handler
}

func (handler *EbpfHandler) HandleData(data []byte) string {
	//Unmarshal the data into a map
	var ebpfDataResponse zkUtilsOtel.EbpfEntryResponse

	err := proto.Unmarshal(data, &ebpfDataResponse)
	if err != nil {
		logger.Debug(ebpfHandlerLogTag, "error unmarshalling data into map ", err)
		return ""
	}

	//Extract trace id and span id
	traceId := ebpfDataResponse.TraceId
	spanId := ebpfDataResponse.SpanId
	ebpfDataForSpan := ebpfDataResponse.EbpfData

	//Save the data into badger
	badgerKey := GetBadgerKey(traceId, spanId)
	logger.Debug(ebpfHandlerLogTag, "Saving data into badger with key: ", badgerKey)
	err = handler.traceBadgerHandler.PutData(badgerKey, ebpfDataForSpan)
	if err != nil {
		logger.Debug(ebpfHandlerLogTag, "Error while saving data into badger with key: ", badgerKey)
		return ""
	}
	//Filter the data based on rules in probes

	//Save the filtered data to DB1

	return "Server received: " + string(data[:])
}

func GetBadgerKey(traceId string, spanId string) string {
	return traceId + "_" + spanId + "_e"
}
