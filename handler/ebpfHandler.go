package handler

import (
	"encoding/json"
	"github.com/gogo/protobuf/proto"
	"github.com/zerok-ai/zk-observer/config"
	"github.com/zerok-ai/zk-observer/stores/badger"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkUtilsOtel "github.com/zerok-ai/zk-utils-go/proto"
	"github.com/zerok-ai/zk-utils-go/socket"
)

var ebpfHandlerLogTag = "ebpfHandler"

type EbpfHandler struct {
	tcpServer          socket.TCPServer
	traceBadgerHandler *badger.TraceBadgerHandler
}

type EbpfDataJson struct {
	ContentType  string `json:"content_type"`
	ReqHeaders   string `json:"req_headers"`
	ReqMethod    string `json:"req_method"`
	ReqPath      string `json:"req_path"`
	ReqBodySize  string `json:"req_body_size"`
	ReqBody      string `json:"req_body"`
	RespHeaders  string `json:"resp_headers"`
	RespStatus   string `json:"resp_status"`
	RespMessage  string `json:"resp_message"`
	RespBodySize string `json:"resp_body_size"`
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	RespBody     string `json:"resp_body"`
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

	errorMessage := "Error while saving data into badger."

	//Unmarshal the data into a json
	var ebpfDataResponse EbpfDataJson
	err := json.Unmarshal(data, &ebpfDataResponse)
	if err != nil {
		logger.Debug(ebpfHandlerLogTag, "error unmarshalling data into map ", err)
		return errorMessage
	}

	//Extract trace id and span id
	traceId := ebpfDataResponse.TraceID
	spanId := ebpfDataResponse.SpanID

	ebpfDataForSpan := &zkUtilsOtel.EbpfEntryDataForSpan{
		ContentType:  ebpfDataResponse.ContentType,
		ReqHeaders:   ebpfDataResponse.ReqHeaders,
		ReqMethod:    ebpfDataResponse.ReqMethod,
		ReqPath:      ebpfDataResponse.ReqPath,
		ReqBodySize:  ebpfDataResponse.ReqBodySize,
		ReqBody:      ebpfDataResponse.ReqBody,
		RespHeaders:  ebpfDataResponse.RespHeaders,
		RespStatus:   ebpfDataResponse.RespStatus,
		RespMessage:  ebpfDataResponse.RespMessage,
		RespBodySize: ebpfDataResponse.RespBodySize,
		RespBody:     ebpfDataResponse.RespBody,
	}

	//Save the data into badger
	logger.Debug(ebpfHandlerLogTag, "Saving data into badger with key: ", traceId, spanId)
	logger.Debug(ebpfHandlerLogTag, "Ebpf data saved is: ", ebpfDataForSpan)
	ebpfDataForSpanBytes, _ := proto.Marshal(ebpfDataForSpan)
	err = handler.traceBadgerHandler.PutEbpfData(traceId, spanId, ebpfDataForSpanBytes)
	if err != nil {
		logger.Debug(ebpfHandlerLogTag, "Error while saving data into badger with key: ", traceId, spanId, err)
		return errorMessage
	}

	return "Success"
}
