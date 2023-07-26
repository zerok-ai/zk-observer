package utils

import (
	"flag"
	"fmt"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"gopkg.in/yaml.v3"
	"os"
)

var LOG_TAG = "utils"

func ProcessArgs(config interface{}) error {
	filePath := flag.String("c", "", "config file path")
	flag.Parse()

	if *filePath == "" {
		zklogger.Error(LOG_TAG, "Please provide a file path using the -c option.")
		return fmt.Errorf("file path provided is empty")
	}

	fileContents, err := os.ReadFile(*filePath)
	if err != nil {
		zklogger.Error(LOG_TAG, "Failed to read file: %s\n", err)
		return err
	}

	err = yaml.Unmarshal(fileContents, config)
	if err != nil {
		zklogger.Error(LOG_TAG, "Failed to unmarshal yaml: %s\n", err)
		return err
	}

	return nil
}
