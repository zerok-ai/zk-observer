package config

import (
	logsConfig "github.com/zerok-ai/zk-utils-go/logs/config"
)

type OtlpConfig struct {
	LogsConfig logsConfig.LogsConfig `yaml:"logs"`
}

func CreateConfig() *OtlpConfig {
	//TODO: Change this code to read from config file.
	logC := logsConfig.LogsConfig{Level: "DEBUG", Color: true}
	return &OtlpConfig{LogsConfig: logC}
}
