package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	logsConfig "github.com/zerok-ai/zk-utils-go/logs/config"
	badgerConfig "github.com/zerok-ai/zk-utils-go/storage/badger/config"
	zkconfig "github.com/zerok-ai/zk-utils-go/storage/redis/config"
	"os"
)

const LOG_TAG = "Config"

type ExceptionConfig struct {
	SyncDuration int `yaml:"syncDuration"`
	BatchSize    int `yaml:"batchSize"`
	Ttl          int `yaml:"ttl"`
}

type ResourceConfig struct {
	SyncDuration int `yaml:"syncDuration"`
	BatchSize    int `yaml:"batchSize"`
	Ttl          int `yaml:"ttl"`
}

type TraceConfig struct {
	SyncDuration int `yaml:"syncDuration"`
	BatchSize    int `yaml:"batchSize"`
	Ttl          int `yaml:"ttl"`
}

type WorkloadConfig struct {
	SyncDuration         int `yaml:"syncDuration"`
	BatchSize            int `yaml:"batchSize"`
	BucketActiveDuration int `yaml:"bucketActiveDuration"`
	Ttl                  int `yaml:"ttl"`
}

type ServiceListConfig struct {
	SyncDuration int `yaml:"syncDuration"`
	BatchSize    int `yaml:"batchSize"`
}

type ScenarioConfig struct {
	SyncDuration int `yaml:"syncDuration"`
}

type OtlpConfig struct {
	Port              string                    `yaml:"port"`
	SetHttpEndpoint   bool                      `yaml:"setHttpEndPoint"`
	SetSpanAttributes bool                      `yaml:"setSpanAttributes"`
	Logs              logsConfig.LogsConfig     `yaml:"logs"`
	Redis             zkconfig.RedisConfig      `yaml:"redis"`
	Badger            badgerConfig.BadgerConfig `yaml:"badger"`
	Traces            TraceConfig               `yaml:"traces"`
	Workloads         WorkloadConfig            `yaml:"workloads"`
	Scenario          ScenarioConfig            `yaml:"scenario"`
	Exception         ExceptionConfig           `yaml:"exception"`
	Resources         ResourceConfig            `yaml:"resources"`
	Services          ServiceListConfig         `yaml:"services"`
}

func CreateConfig(configPath string) *OtlpConfig {
	var cfg OtlpConfig

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		logger.Error(LOG_TAG, err)
		os.Exit(2)
	}
	return &cfg
}
