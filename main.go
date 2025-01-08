package main

import (
	"fmt"
	"os"

	"hcloud-k8s/cmd/hcloud-k8s"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
} 