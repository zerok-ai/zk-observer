package config

import (
	logsConfig "github.com/zerok-ai/zk-utils-go/logs/config"
)

type RedisConfig struct {
	Host          string         `yaml:"host" env:"ZK_REDIS_HOST" env-description:"Database host"`
	Password      string         `yaml:"password" env:"ZK_REDIS_PASSWORD" env-description:"Database password"`
	Port          string         `yaml:"port" env-description:"Database port"`
	DBs           map[string]int `yaml:"dbs" env-description:"Database to load"`
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
