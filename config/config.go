package config

import (
	logsConfig "github.com/zerok-ai/zk-utils-go/logs/config"
)

type RedisConfig struct {
	Host          string         `yaml:"host" env:"REDIS_HOST" env-description:"Database host"`
	Port          string         `yaml:"port" env:"REDIS_PORT" env-description:"Database port"`
	DBs           map[string]int `yaml:"dbs" env:"REDIS_DB" env-description:"Database to load"`
	Ttl           int            `yaml:"ttl"`
	SyncDuration  int            `yaml:"syncDuration"`
	TimerDuration int            `yaml:"timerDuration"`
	BatchSize     int            `yaml:"batchSize"`
}

type OtlpConfig struct {
	Port              string                `yaml:"port"`
	SetHttpEndpoint   bool                  `yaml:"setHttpEndPoint"`
	SetSpanAttributes bool                  `yaml:"setSpanAttributes"`
	Logs              logsConfig.LogsConfig `yaml:"logs"`
	Redis             RedisConfig           `yaml:"redis"`
}

func CreateConfig() *OtlpConfig {
	//TODO: Change this code to read from config file.
	logC := logsConfig.LogsConfig{Level: "DEBUG", Color: true}
	return &OtlpConfig{Logs: logC}
}
