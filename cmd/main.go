package main

import (
	"context"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/handler"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	zkconfig "github.com/zerok-ai/zk-utils-go/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	pb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"net"
)

var mainLogTag = "main"

type grpcServer struct {
	pb.UnimplementedTraceServiceServer
	traceHandler *handler.TraceHandler
}

func (s *grpcServer) Export(context context.Context, req *pb.ExportTraceServiceRequest) (*pb.ExportTraceServiceResponse, error) {
	logger.Debug(mainLogTag, "Export method invoked.")
	peer, ok := peer.FromContext(context)
	remoteAddr := ""
	if ok {
		remoteAddr = peer.Addr.String()
		remoteAddr = utils.GetClientIP(remoteAddr)
	}
	s.traceHandler.ProcessTraceData(req.ResourceSpans, remoteAddr)
	err := s.traceHandler.PushDataToRedis()
	if err != nil {
		logger.Error(mainLogTag, "Error while pushing data to redis ", err)
	}
	return &pb.ExportTraceServiceResponse{}, nil
}

func main() {

	otlpConfig := config.CreateConfig()

	if err := zkconfig.ProcessArgs(otlpConfig); err != nil {
		logger.Error(mainLogTag, "Unable to process wsp client config. Stopping wsp client.")
		return
	}

	logger.Init(otlpConfig.Logs)

	traceHandler, err := handler.NewTraceHandler(otlpConfig)

	if err != nil {
		logger.Error(mainLogTag, "Error while creating traceHandler:", err)
		return
	}

	logger.Debug(mainLogTag, "Starting grpc server.")

	//Creating grpc server
	listener, err := net.Listen("tcp", ":4317")
	if err != nil {
		logger.Error(mainLogTag, "Error while creating grpc listener:", err)
		return
	}
	s := grpc.NewServer()
	pb.RegisterTraceServiceServer(s, &grpcServer{traceHandler: traceHandler})
	go s.Serve(listener)

	logger.Debug(mainLogTag, "Started grpc server.")

	//Creating http/protobuf server

	app := newApp()
	irisConfig := iris.WithConfiguration(iris.Configuration{
		DisablePathCorrection: true,
		LogLevel:              otlpConfig.Logs.Level,
	})

	app.Post("/v1/traces", traceHandler.ServeHTTP)

	err = app.Run(iris.Addr(":"+otlpConfig.Port), irisConfig)

	if err != nil {
		logger.Error(mainLogTag, "Error starting the server:", err)
	}

}

func newApp() *iris.Application {
	app := iris.Default()

	crs := func(ctx iris.Context) {
		ctx.Header("Access-Control-Allow-Credentials", "true")

		if ctx.Method() == iris.MethodOptions {
			ctx.Header("Access-Control-Methods", "POST")

			ctx.Header("Access-Control-Allow-Headers",
				"Access-Control-Allow-Origin,Content-Type")

			ctx.Header("Access-Control-Max-Age",
				"86400")

			ctx.StatusCode(iris.StatusNoContent)
			return
		}

		ctx.Next()
	}
	app.UseRouter(crs)
	app.AllowMethods(iris.MethodOptions)

	return app
}
