package handler

import (
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-utils-go/socket"
)

type EbpfHandler struct {
	tcpServer socket.TCPServer
}

func CreateAndStartEbpfHandler(config *config.OtlpConfig) *EbpfHandler {
	handler := EbpfHandler{}
	tcpServer := socket.CreateTCPServer(config.TcpServerConfig, handler.HandleData)
	handler.tcpServer = *tcpServer
	tcpServer.Start()
	return &handler
}

func (handler *EbpfHandler) HandleData(data []byte) string {
	return "Server received: " + string(data[:])
}
