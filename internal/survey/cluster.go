package survey

import (
	"context"
	"github.com/AlecAivazis/survey/v2"
	"hcloud-k8s/internal/hetzner"
)

// ClusterConfig holds the configuration for a Kubernetes cluster
type ClusterConfig struct {
	ClusterName       string
	Region           string
	MasterNodeCount  int
	WorkerNodeCount  int
	NodeType         string
	KubernetesVersion string
	HetznerToken     string
}

// GetClusterConfig collects cluster configuration through interactive prompts
func GetClusterConfig(ctx context.Context) (*ClusterConfig, error) {
	config := &ClusterConfig{}
	
	// Get token first
	if err := survey.AskOne(&survey.Password{
		Message: "Enter your Hetzner Cloud API token:",
	}, &config.HetznerToken); err != nil {
		return nil, err
	}

	// Create client with token
	client := hetzner.New(config.HetznerToken)

	// Get available options
	regions, err := client.GetLocations(ctx)
	if err != nil {
		return nil, err
	}

	nodeTypes, err := client.GetServerTypes(ctx)
	if err != nil {
		return nil, err
	}

	questions := []*survey.Question{
		{
			Name: "clusterName",
			Prompt: &survey.Input{
				Message: "What is the name of your cluster?",
				Default: "my-cluster",
			},
		},
		{
			Name: "region",
			Prompt: &survey.Select{
				Message: "Choose a region:",
				Options: regions,
			},
		},
		{
			Name: "masterNodeCount",
			Prompt: &survey.Input{
				Message: "How many master nodes?",
				Default: "3",
			},
		},
		{
			Name: "workerNodeCount",
			Prompt: &survey.Input{
				Message: "How many worker nodes?",
				Default: "3",
			},
		},
		{
			Name: "nodeType",
			Prompt: &survey.Select{
				Message: "Choose node type:",
				Options: nodeTypes,
			},
		},
		{
			Name: "kubernetesVersion",
			Prompt: &survey.Select{
				Message: "Choose Kubernetes version:",
				Options: []string{"1.27", "1.26", "1.25"},
			},
		},
	}

	if err := survey.Ask(questions, config); err != nil {
		return nil, err
	}

	return config, nil
} 