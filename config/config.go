package config

import (
	logsConfig "github.com/zerok-ai/zk-utils-go/logs/config"
	zkconfig "github.com/zerok-ai/zk-utils-go/storage/redis/config"
)

type ExceptionConfig struct {
	SyncDuration int `yaml:"syncDuration"`
	BatchSize    int `yaml:"batchSize"`
}

type ResourceConfig struct {
	SyncDuration int `yaml:"syncDuration"`
	BatchSize    int `yaml:"batchSize"`
}

type TraceConfig struct {
	SyncDuration int `yaml:"syncDuration"`
	BatchSize    int `yaml:"batchSize"`
	Ttl          int `yaml:"ttl"`
}

type WorkloadConfig struct {
	SyncDuration int `yaml:"syncDuration"`
	BatchSize    int `yaml:"batchSize"`
	Ttl          int `yaml:"ttl"`
}

type ScenarioConfig struct {
	SyncDuration int `yaml:"syncDuration"`
}

type OtlpConfig struct {
	Port              string                `yaml:"port"`
	SetHttpEndpoint   bool                  `yaml:"setHttpEndPoint"`
	SetSpanAttributes bool                  `yaml:"setSpanAttributes"`
	Logs              logsConfig.LogsConfig `yaml:"logs"`
	Redis             zkconfig.RedisConfig  `yaml:"redis"`
	Traces            TraceConfig           `yaml:"traces"`
	Workloads         WorkloadConfig        `yaml:"workloads"`
	Scenario          ScenarioConfig        `yaml:"scenario"`
	Exception         ExceptionConfig       `yaml:"exceptions"`
	Resources         ResourceConfig        `yaml:"resources"`
}

func CreateConfig() *OtlpConfig {
	//TODO: Change this code to read from config file.
	logC := logsConfig.LogsConfig{Level: "DEBUG", Color: true}
	return &OtlpConfig{Logs: logC}
}
