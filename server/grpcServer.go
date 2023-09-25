package server

import (
	"context"
	"github.com/zerok-ai/zk-otlp-receiver/handler"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	pb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc/peer"
)

var grpcServerTag = "grpcServer"

type GrpcServer struct {
	pb.UnimplementedTraceServiceServer
	TraceHandler *handler.TraceHandler
}

func (s *GrpcServer) Export(context context.Context, req *pb.ExportTraceServiceRequest) (*pb.ExportTraceServiceResponse, error) {
	logger.Debug(grpcServerTag, "Export method invoked.")
	peer, ok := peer.FromContext(context)
	remoteAddr := ""
	if ok {
		remoteAddr = peer.Addr.String()
		remoteAddr = utils.GetClientIP(remoteAddr)
	}
	s.TraceHandler.ProcessTraceData(req.ResourceSpans, remoteAddr)
	err := s.TraceHandler.PushDataToRedis()
	if err != nil {
		logger.Error(grpcServerTag, "Error while pushing data to redis ", err)
	}
	return &pb.ExportTraceServiceResponse{}, nil
}
