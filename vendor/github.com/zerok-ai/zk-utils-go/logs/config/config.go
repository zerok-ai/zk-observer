package config

type LogsConfig struct {
	Color bool   `yaml:"color" env-description:"Auth token expiry in seconds"`
	Level string `yaml:"level" env-description:"Auth token expiry in seconds"`
}
