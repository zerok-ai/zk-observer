package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/zerok-ai/zk-observer/config"
	"github.com/zerok-ai/zk-observer/stores/badger"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkUtilsOtel "github.com/zerok-ai/zk-utils-go/proto"
	"github.com/zerok-ai/zk-utils-go/socket"
	"regexp"
	"strings"
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
	logger.Debug(ebpfHandlerLogTag, "Creating ebpf handler on config ", config.TcpServerConfig)
	tcpServer := socket.CreateTCPServer(config.TcpServerConfig, handler.HandleData)
	handler.tcpServer = *tcpServer
	handler.traceBadgerHandler = traceBadgerHandler
	tcpServer.Start()
	return &handler
}

func (handler *EbpfHandler) HandleData(data []byte) string {

	errorMessage := "Error while saving data into badger."

	//Removing first five chars which contain messageid
	jsonData := data[5:]
	jsonString := string(jsonData)
	cleanedJsonString := strings.ReplaceAll(jsonString, "\x00", "")
	traces := splitTraces(cleanedJsonString)
	for _, trace := range traces {
		handler.handleSingleTrace(trace, errorMessage)
	}
	return "Success"
}

func (handler *EbpfHandler) handleSingleTrace(cleanedJsonString string, errorMessage string) {
	//Unmarshal the data into a json
	var ebpfDataResponse EbpfDataJson
	err := json.Unmarshal([]byte(cleanedJsonString), &ebpfDataResponse)
	if err != nil {
		logger.Debug(ebpfHandlerLogTag, "error unmarshalling data into map ", err)
		return
	}

	traceParent, err := extractTraceParent(ebpfDataResponse.ReqHeaders)
	if err != nil || traceParent == "" {
		logger.Debug(ebpfHandlerLogTag, "Could not extract traceParent, ignoring the trace.")
		return
	}

	logger.Debug(ebpfHandlerLogTag, "TraceParent: ", traceParent)

	//Extract trace id and span id
	traceId, spanId, err := getTraceIdAndSpanIdFromTraceParent(traceParent)
	if err != nil || traceId == "" || spanId == "" {
		logger.Debug(ebpfHandlerLogTag, "Could not extract traceId and spanId, ignoring the trace.")
		return
	}

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
	ebpfDataForSpanBytes, _ := proto.Marshal(ebpfDataForSpan)
	err = handler.traceBadgerHandler.PutEbpfData(traceId, spanId, ebpfDataForSpanBytes)
	if err != nil {
		logger.Debug(ebpfHandlerLogTag, "Error while saving data into badger with key: ", traceId, spanId, err)
	}
}

func getTraceIdAndSpanIdFromTraceParent(key string) (string, string, error) {

	splitKey := strings.Split(key, "-")
	if len(splitKey) < 3 {
		return "", "", fmt.Errorf("invalid traceParent value")
	}
	traceId := splitKey[1]
	spanId := splitKey[2]

	return traceId, spanId, nil
}

// Function to extract the traceParent value from req_headers string
func extractTraceParent(input string) (string, error) {
	// Regex pattern to match the traceParent
	pattern := `"traceparent":"([^"]+)"`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	matches := re.FindStringSubmatch(input)

	// Check if a match was found
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("traceParent value not found")
}

func splitTraces(traces string) []string {
	re := regexp.MustCompile(`\}.{4};\{`)
	parts := re.Split(traces, -1)

	if len(parts) == 1 {
		return parts
	}

	//Adding end bracket to the first part
	parts[0] = parts[0] + "}"

	//Adding start bracket to the last part
	parts[len(parts)-1] = "{" + parts[len(parts)-1]

	//Add the start and end brackets to the other parts
	for i := 1; i < len(parts)-1; i++ {
		parts[i] = "{" + parts[i] + "}"
	}
	return parts
}
