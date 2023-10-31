package server

import (
	"context"
	"github.com/zerok-ai/zk-otlp-receiver/handler"
	pb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

var grpcServerTag = "grpcServer"

type GrpcServer struct {
	pb.UnimplementedTraceServiceServer
	TraceHandler *handler.TraceHandler
}

func (s *GrpcServer) Export(context context.Context, req *pb.ExportTraceServiceRequest) (*pb.ExportTraceServiceResponse, error) {
	s.TraceHandler.ProcessTraceData(req.ResourceSpans)
	s.TraceHandler.PushDataToRedis()
	return &pb.ExportTraceServiceResponse{}, nil
}
