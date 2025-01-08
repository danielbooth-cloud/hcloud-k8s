package main

import (
	"os"
	"hcloud-k8s/cmd/hcloud-k8s"
	"hcloud-k8s/internal/logging"
)

func main() {
	logging.InitLogger(logging.LevelInfo)
	logger := logging.GetLogger("main")

	if err := cmd.Execute(); err != nil {
		logger.Error("failed to execute command", "error", err)
		os.Exit(1)
	}
} 