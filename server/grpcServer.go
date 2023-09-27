package server

import (
	"context"
	"github.com/zerok-ai/zk-otlp-receiver/handler"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	pb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

var grpcServerTag = "grpcServer"

type GrpcServer struct {
	pb.UnimplementedTraceServiceServer
	TraceHandler *handler.TraceHandler
}

func (s *GrpcServer) Export(context context.Context, req *pb.ExportTraceServiceRequest) (*pb.ExportTraceServiceResponse, error) {
	s.TraceHandler.ProcessTraceData(req.ResourceSpans)
	err := s.TraceHandler.PushDataToRedis()
	if err != nil {
		logger.Error(grpcServerTag, "Error while pushing data to redis ", err)
	}
	return &pb.ExportTraceServiceResponse{}, nil
}