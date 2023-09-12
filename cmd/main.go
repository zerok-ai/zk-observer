package main

import (
	"github.com/kataras/iris/v12"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	"github.com/zerok-ai/zk-otlp-receiver/handler"
	"github.com/zerok-ai/zk-otlp-receiver/utils"
	zkconfig "github.com/zerok-ai/zk-utils-go/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
)

var LOG_TAG = "main"

func main() {

	otlpConfig := config.CreateConfig()

	if err := zkconfig.ProcessArgs(otlpConfig); err != nil {
		logger.Error(LOG_TAG, "Unable to process wsp client config. Stopping wsp client.")
		return
	}

	logger.Init(otlpConfig.Logs)

	redisHandler, err := utils.NewRedisHandler(&otlpConfig.Redis)
	if err != nil {
		logger.Error(LOG_TAG, "Error while creating redis handler:", err)
		return
	}

	traceHandler := handler.NewTraceHandler(redisHandler, otlpConfig)

	app := newApp()
	irisConfig := iris.WithConfiguration(iris.Configuration{
		DisablePathCorrection: true,
		LogLevel:              otlpConfig.Logs.Level,
	})

	app.Post("/v1/traces", traceHandler.ServeHTTP)

	err = app.Run(iris.Addr(":"+otlpConfig.Port), irisConfig)

	if err != nil {
		logger.Error(LOG_TAG, "Error starting the server:", err)
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
