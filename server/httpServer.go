package server

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/kataras/iris/v12"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zerok-ai/zk-observer/config"
	"github.com/zerok-ai/zk-observer/handler"
	promMetrics "github.com/zerok-ai/zk-observer/metrics"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"net/http"
	"os"
	"time"
)

var httpServerTag = "httpServer"
var httpServerLogTag = "httpServer"
var podIp = os.Getenv("POD_IP")

type HTTPServer struct {
	app *iris.Application
}

func NewHTTPServer() *HTTPServer {
	return &HTTPServer{
		app: newApp(),
	}
}

func (s *HTTPServer) ConfigureRoutes(traceHandler *handler.TraceHandler) {
	s.app.Get("/metrics", iris.FromStd(promhttp.Handler()))
	s.app.Get("/debug/vars", iris.FromStd(http.DefaultServeMux))
	s.app.Get("/healthz", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusOK)
	})
	s.app.Post("/v1/traces", traceHandler.ServeHTTP)
	configureBadgerGetStreamAPI(s.app, traceHandler)
}

func (s *HTTPServer) Run(otlpConfig config.OtlpConfig) error {
	srv := &http.Server{
		Addr:         ":" + otlpConfig.Port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	irisConfig := iris.WithConfiguration(iris.Configuration{
		DisablePathCorrection: true,
		LogLevel:              otlpConfig.Logs.Level,
	})

	return s.app.Run(iris.Server(srv), irisConfig)
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

func configureBadgerGetStreamAPI(app *iris.Application, traceHandler *handler.TraceHandler) {
	app.Post("get-trace-data", func(ctx iris.Context) {

		logger.Fatal(httpServerLogTag, "Request Received to get span data from SM")

		var inputList []string
		promMetrics.TotalFetchRequestsFromSM.WithLabelValues(podIp).Inc()
		// Read the JSON input containing the list of strings
		if err := ctx.ReadJSON(&inputList); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			err := ctx.JSON(iris.Map{"error": "Invalid JSON input"})
			logger.Info(httpServerLogTag, fmt.Sprintf("Request Received to get span data : %s", inputList))
			if err != nil {
				promMetrics.TotalFetchRequestsFromSMError.WithLabelValues(podIp).Inc()
				logger.Error(httpServerLogTag, "Invalid request format for fetching badger data for trace prefix list ", err)
				return
			}
			return
		}

		//total traces span data requested from receiver
		promMetrics.TotalTracesSpanDataRequestedFromReceiver.WithLabelValues(podIp).Add(float64(len(inputList)))

		data, err2 := traceHandler.GetBulkDataFromBadgerForPrefix(inputList)
		if err2 != nil {
			promMetrics.TotalFetchRequestsFromSMError.WithLabelValues(podIp).Inc()
			logger.Error(httpServerLogTag, fmt.Sprintf("Unable to fetch data from badger for tracePrefixList: %s", inputList), err2)
			ctx.StatusCode(iris.StatusInternalServerError)
			return
		}
		ctx.StatusCode(iris.StatusOK)

		protoData, err := proto.Marshal(data)
		if err != nil {
			promMetrics.TotalFetchRequestsFromSMError.WithLabelValues(podIp).Inc()
			logger.Error(httpServerLogTag, fmt.Sprintf("Unable to fetch data from badger for tracePrefixList: %s", inputList), err)
			return
		}
		ctx.ContentType("application/octet-stream")
		_, err = ctx.Write(protoData)
		if err != nil {
			logger.Error(httpServerLogTag, fmt.Sprintf("Unable to fetch data from badger for trace prefix list: %s", inputList), err)
			return
		}
		promMetrics.TotalFetchRequestsFromSMSuccess.WithLabelValues(podIp).Inc()

	}).Describe("Badger Data Fetch API")
}
