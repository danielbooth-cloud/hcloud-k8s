package survey

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"hcloud-k8s/internal/hetzner"
	"hcloud-k8s/internal/kubernetes"
	"regexp"
	"hcloud-k8s/internal/errors"
)

// ClusterConfig holds the configuration for a Kubernetes cluster
type ClusterConfig struct {
	ClusterName           string
	Region               string
	MasterNodeCount      int
	WorkerNodeCount      int
	NodeType             string
	KubernetesVersion    string
	KubernetesDistribution string
	HetznerToken         string
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
	}

	if err := survey.Ask(questions, config); err != nil {
		return nil, err
	}

	// Get Kubernetes distribution after basic cluster config
	if err := survey.AskOne(&survey.Select{
		Message: "Choose Kubernetes distribution:",
		Options: kubernetes.GetDistributions(),
	}, &config.KubernetesDistribution); err != nil {
		return nil, err
	}

	// Get available versions for selected distribution
	versions, err := kubernetes.GetVersions(ctx, kubernetes.Distribution(config.KubernetesDistribution))
	if err != nil {
		return nil, fmt.Errorf("failed to get versions for %s: %v", config.KubernetesDistribution, err)
	}

	if err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Choose %s version:", config.KubernetesDistribution),
		Options: versions,
	}, &config.KubernetesVersion); err != nil {
		return nil, err
	}

	return config, nil
}

func validateClusterName(name string) error {
	if len(name) < 3 || len(name) > 63 {
		return errors.NewValidationError("clusterName", "must be between 3 and 63 characters")
	}
	
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`).MatchString(name) {
		return errors.NewValidationError("clusterName", "must contain only lowercase letters, numbers, and hyphens, and must start and end with a letter or number")
	}
	
	return nil
}

func validateNodeCount(count int, nodeType string) error {
	if count < 1 {
		return errors.NewValidationError(nodeType+"NodeCount", "must be at least 1")
	}
	if count > 10 {
		return errors.NewValidationError(nodeType+"NodeCount", "cannot exceed 10")
	}
	return nil
} 