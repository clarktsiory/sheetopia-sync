package config

import (
	"fmt"
	"os"
	"strconv"
)

var DataDir string
var Port int

func Load() error {
	DataDir = os.Getenv("DATA_DIR")
	if DataDir == "" {
		DataDir = "data"
	}

	err := os.MkdirAll(DataDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create data dir: %s", err)
	}

	portStr := os.Getenv("PORT")
	Port = 8080
	if portStr != "" {
		Port, err = strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %s", err)
		}
	}
	return nil
}
