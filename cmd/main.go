package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/handler"
	"github.com/zerok-ai/zk-otlp-receiver/server"
	zkconfig "github.com/zerok-ai/zk-utils-go/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
	pb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"net"
	"os"
)

var mainLogTag = "main"
var ctx = context.Background()

type Args struct {
	ConfigPath string
}

func main() {
	var cfg Args
	args := ProcessArgs(&cfg)

	otlpConfig := config.CreateConfig(args.ConfigPath)

	if err := zkconfig.ProcessArgs(otlpConfig); err != nil {
		logger.Error(mainLogTag, "Unable to process wsp client config. Stopping wsp client.")
		return
	}

	logger.Init(otlpConfig.Logs)
	storeFactory := *stores.GetStoreFactory(otlpConfig.Redis, ctx)

	traceHandler, err := handler.NewTraceHandler(otlpConfig, storeFactory)

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
	pb.RegisterTraceServiceServer(s, &server.GrpcServer{TraceHandler: traceHandler})
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

func ProcessArgs(cfg interface{}) Args {
	// ProcessArgs processes and handles CLI arguments
	var a Args

	f := flag.NewFlagSet("Example server", 1)
	f.StringVar(&a.ConfigPath, "c", "config.yaml", "Path to configuration file")

	fu := f.Usage
	f.Usage = func() {
		fu()
		envHelp, _ := cleanenv.GetDescription(cfg, nil)
		fmt.Fprintln(f.Output())
		fmt.Fprintln(f.Output(), envHelp)
	}

	f.Parse(os.Args[1:])
	return a
}
