package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/zerok-ai/zk-observer/config"
	"github.com/zerok-ai/zk-observer/handler"
	promMetrics "github.com/zerok-ai/zk-observer/metrics"
	"github.com/zerok-ai/zk-observer/server"
	"github.com/zerok-ai/zk-observer/stores/badger"
	zkconfig "github.com/zerok-ai/zk-utils-go/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
	"os"
)

var mainLogTag = "main"
var ctx = context.Background()
var podIp = os.Getenv("POD_IP")

type Args struct {
	ConfigPath string
}

// register collector method
func init() {
	prometheus.MustRegister(promMetrics.BadgerCollector(""))
}

func main() {
	var appArgs Args
	args := ProcessArgs(&appArgs)

	otlpConfig := config.CreateConfig(args.ConfigPath)
	if err := zkconfig.ProcessArgs(otlpConfig); err != nil {
		logger.Error(mainLogTag, "Unable to process service config.")
		return
	}

	logger.Init(otlpConfig.Logs)
	storeFactory := *stores.GetStoreFactory(otlpConfig.Redis, ctx)

	traceBadgerHandler, err := badger.NewTracesBadgerHandler(otlpConfig)
	if err != nil {
		logger.Error(mainLogTag, "Error while creating badger handler:", err)
		return
	}

	traceHandler, err := handler.NewTraceHandler(otlpConfig, storeFactory, traceBadgerHandler)

	if err != nil {
		logger.Error(mainLogTag, "Error while creating traceHandler:", err)
		return
	}

	//Creating grpc server
	//listener, err := net.Listen("tcp", ":4317")
	//if err != nil {
	//	logger.Error(mainLogTag, "Error while creating grpc listener:", err)
	//	return
	//}
	//s := grpc.NewServer()
	//pb.RegisterTraceServiceServer(s, &server.GrpcServer{TraceHandler: traceHandler})
	//go s.Serve(listener)

	logger.Debug(mainLogTag, "Starting ebpf socket.")
	go handler.CreateAndStartEbpfHandler(otlpConfig, traceBadgerHandler)

	//Creating http/protobuf server
	// Instantiate the HTTPServer
	httpServer := server.NewHTTPServer()
	// Configure routes and pass the traceHandler
	httpServer.ConfigureRoutes(traceHandler)
	// Run the HTTP server with the specified port and configs
	err = httpServer.Run(*otlpConfig)
	if err != nil {
		logger.Error(mainLogTag, "Error starting the server:", err)
	}
	logger.Debug(mainLogTag, "Started http server.")

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
