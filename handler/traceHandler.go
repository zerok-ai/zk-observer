package handler

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	v1 "github.com/zerok-ai/zk-otlp-receiver/proto/go.opentelemetry.io/proto/otlp/trace/v1"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"io"
	"net/http"
)

var TRACE_LOG_TAG = "TraceHandler"

func HandleTraceRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Unmarshal the protobuf message from the request body
	var traceData v1.TracesData
	err = proto.Unmarshal(body, &traceData)
	if err != nil {
		http.Error(w, "Failed to unmarshal proto data", http.StatusBadRequest)
		return
	}

	// Print the proto data received
	logger.Debug(TRACE_LOG_TAG, "Received Trace:\n%v\n", traceData)

	// Respond to the client
	w.WriteHeader(http.StatusOK)
	_, err = fmt.Fprintln(w, "Trace data received successfully!")
	if err != nil {
		return
	}
}
