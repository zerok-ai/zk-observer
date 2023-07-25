package logger

import (
	"github.com/zerok-ai/zk-utils-go/logs/config"
	"log"
)

type ZkLogger interface {
	Debug(tag string, messages ...any)
	Info(tag string, messages ...any)
	Warn(tag string, messages ...any)
	Err(tag string, messages ...any)
	Fatal(tag string, messages ...any)
	Log(level logLevel, tag string, messages ...any)
}

var minLogLevel = _DEBUG_LEVEL
var addColors = true

func Init(logsConfig config.LogsConfig) {
	switch {
	case logsConfig.Level == "DEBUG":
		minLogLevel = _DEBUG_LEVEL
	case logsConfig.Level == "INFO":
		minLogLevel = _INFO_LEVEL
	case logsConfig.Level == "WARN":
		minLogLevel = _WARN_LEVEL
	case logsConfig.Level == "ERROR":
		minLogLevel = _ERROR_LEVEL
	case logsConfig.Level == "FATAL":
		minLogLevel = _FATAL_LEVEL
	}

	addColors = logsConfig.Color
}

type logLevel struct {
	Value int
	Label string
	Color string
}

var (
	_DEBUG_LEVEL = logLevel{Value: 1, Label: "DEBUG", Color: colorDebug}
	_INFO_LEVEL  = logLevel{Value: 2, Label: "INFO", Color: colorInfo}
	_WARN_LEVEL  = logLevel{Value: 3, Label: "WARN", Color: colorWarn}
	_ERROR_LEVEL = logLevel{Value: 4, Label: "ERROR", Color: colorError}
	_FATAL_LEVEL = logLevel{Value: 5, Label: "FATAL", Color: colorFatal}
)

var (
	colorReset  string = "\033[0m"
	colorRed    string = "\033[31m"
	colorGreen  string = "\033[32m"
	colorYellow string = "\033[33m"
	colorBlue   string = "\033[34m"
	colorPurple string = "\033[35m"
	colorCyan   string = "\033[36m"
	colorWhite  string = "\033[37m"

	colorInfo  string = colorBlue
	colorError string = colorRed
	colorWarn  string = colorYellow
	colorDebug string = colorReset
	colorFatal string = colorPurple
)

func Debug(tag string, messages ...any) {
	Log(_DEBUG_LEVEL, tag, messages...)
}

func Info(tag string, messages ...any) {
	Log(_INFO_LEVEL, tag, messages...)
}

func Warn(tag string, messages ...any) {
	Log(_WARN_LEVEL, tag, messages...)
}

func Error(tag string, messages ...any) {
	Log(_ERROR_LEVEL, tag, messages...)
}

func Fatal(tag string, messages ...any) {
	Log(_FATAL_LEVEL, tag, messages...)
}

func Log(level logLevel, tag string, messages ...any) {
	var newMessages []interface{}
	if minLogLevel.Value <= level.Value {
		if addColors {
			newMessages = append([]interface{}{level.Color, "[" + level.Label + "]"})
		} else {
			newMessages = append([]interface{}{"[" + level.Label + "]"})
		}
		newMessages = append(newMessages, tag, "|")
		newMessages = append(newMessages, messages...)
		newMessages = append(newMessages, colorReset)
		log.Println(newMessages...)

		// messages = append([]interface{}{ "["+level.Label+"]", tag, "|" }, messages...)
		// log.Println(messages...)
	}
}
