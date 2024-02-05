package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/kataras/iris/v12"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zerok-ai/zk-observer/config"
	"github.com/zerok-ai/zk-observer/handler"
	promMetrics "github.com/zerok-ai/zk-observer/metrics"
	"github.com/zerok-ai/zk-observer/stores/badger"
	zkconfig "github.com/zerok-ai/zk-utils-go/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-utils-go/storage/redis/stores"
	"net/http"
	"os"
	"time"
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

	//Starting tcp socket sever for receiving ebpf data
	logger.Debug(mainLogTag, "Starting tcp server.")
	go handler.CreateAndStartEbpfHandler(otlpConfig, traceBadgerHandler)

	logger.Debug(mainLogTag, "Starting grpc server.")

	//Creating grpc server
	//listener, err := net.Listen("tcp", ":4317")
	//if err != nil {
	//	logger.Error(mainLogTag, "Error while creating grpc listener:", err)
	//	return
	//}
	//s := grpc.NewServer()
	//pb.RegisterTraceServiceServer(s, &server.GrpcServer{TraceHandler: traceHandler})
	//go s.Serve(listener)

	logger.Debug(mainLogTag, "Started grpc server.")

	//Creating http/protobuf server

	app := newApp()
	irisConfig := iris.WithConfiguration(iris.Configuration{
		DisablePathCorrection: true,
		LogLevel:              otlpConfig.Logs.Level,
	})

	app.Get("/metrics", iris.FromStd(promhttp.Handler()))

	app.Get("/prefixget", func(ctx iris.Context) {
		//Get query param from iris context
		prefix := ctx.URLParam("prefix")
		valuesMap, err := traceBadgerHandler.PrefixGet(prefix)
		//log error
		if err != nil {
			logger.Error(mainLogTag, "Error while getting data for prefix from badger:", err)
			return
		}
		//log value
		logger.Info(mainLogTag, fmt.Sprintf("Value for prefix %s is %s", prefix, valuesMap))
	})

	app.Get("/getvar", func(ctx iris.Context) {
		//Get query param from iris context
		key := ctx.URLParam("key")
		value, err := traceBadgerHandler.GetData(key)
		//log error
		if err != nil {
			logger.Error(mainLogTag, "Error while getting data from badger:", err)
			return
		}
		//log value
		logger.Info(mainLogTag, fmt.Sprintf("Value for key %s is %s", key, value))
	})

	// Define a route to expose expvar data
	app.Get("/debug/vars", iris.FromStd(http.DefaultServeMux))
	app.Get("/healthz", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusOK)
	})

	app.Post("/v1/traces", traceHandler.ServeHTTP)
	configureBadgerGetStreamAPI(app, traceHandler)

	// Start the server with timeouts
	srv := &http.Server{
		Addr:         ":" + otlpConfig.Port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	err = app.Run(iris.Server(srv), irisConfig)

	if err != nil {
		logger.Error(mainLogTag, "Error starting the server:", err)
	}

	//badger
	//newBadgerApp := newBadgerApp()

	//err = newBadgerApp.Run(iris.Addr(":"+"8047"), irisConfig)
	//if err != nil {
	//	logger.Error(mainLogTag, "Error starting the server:", err)
	//}

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

func newBadgerApp() *iris.Application {
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

func configureBadgerGetStreamAPI(app *iris.Application, traceHandler *handler.TraceHandler) {
	app.Post("get-trace-data", func(ctx iris.Context) {

		logger.Fatal(mainLogTag, "Request Received to get span data from SM")

		var inputList []string
		promMetrics.TotalFetchRequestsFromSM.WithLabelValues(podIp).Inc()
		// Read the JSON input containing the list of strings
		if err := ctx.ReadJSON(&inputList); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			err := ctx.JSON(iris.Map{"error": "Invalid JSON input"})
			logger.Info(mainLogTag, fmt.Sprintf("Request Received to get span data : %s", inputList))
			if err != nil {
				promMetrics.TotalFetchRequestsFromSMError.WithLabelValues(podIp).Inc()
				logger.Error(mainLogTag, "Invalid request format for fetching badger data for trace prefix list ", err)
				return
			}
			return
		}

		//total traces span data requested from receiver
		promMetrics.TotalTracesSpanDataRequestedFromReceiver.WithLabelValues(podIp).Add(float64(len(inputList)))

		data, err2 := traceHandler.GetBulkDataFromBadgerForPrefix(inputList)
		if err2 != nil {
			promMetrics.TotalFetchRequestsFromSMError.WithLabelValues(podIp).Inc()
			logger.Error(mainLogTag, fmt.Sprintf("Unable to fetch data from badger for tracePrefixList: %s", inputList), err2)
			ctx.StatusCode(iris.StatusInternalServerError)
			return
		}
		ctx.StatusCode(iris.StatusOK)
		//jsonData, err := json.Marshal(data)
		//if err != nil {
		//	logger.Error(mainLogTag, fmt.Sprintf("Unable to fetch data from badger for tracePrefixList: %s", inputList), err)
		//	return
		//}

		protoData, err := proto.Marshal(data)
		if err != nil {
			promMetrics.TotalFetchRequestsFromSMError.WithLabelValues(podIp).Inc()
			logger.Error(mainLogTag, fmt.Sprintf("Unable to fetch data from badger for tracePrefixList: %s", inputList), err)
			return
		}
		ctx.ContentType("application/octet-stream")
		_, err = ctx.Write(protoData)
		if err != nil {
			logger.Error(mainLogTag, fmt.Sprintf("Unable to fetch data from badger for trace prefix list: %s", inputList), err)
			return
		}
		promMetrics.TotalFetchRequestsFromSMSuccess.WithLabelValues(podIp).Inc()

	}).Describe("Badger Data Fetch API")
}
