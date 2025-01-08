package cmd

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/spf13/cobra"
)

type ClusterConfig struct {
	ClusterName       string
	Region           string
	NodeCount        int
	NodeType         string
	KubernetesVersion string
	HetznerToken     string
}

type HetznerOptions struct {
	Regions    []string
	NodeTypes  []string
	K8sVersions []string
}

func createQuestion(name, message string, prompt survey.Prompt) *survey.Question {
	return &survey.Question{
		Name:   name,
		Prompt: prompt,
	}
}

func createSelectQuestion(name, message string, options []string) *survey.Question {
	return createQuestion(name, message, &survey.Select{
		Message: message,
		Options: options,
		Default: options[0],
	})
}

func createInputQuestion(name, message, defaultValue string) *survey.Question {
	return createQuestion(name, message, &survey.Input{
		Message: message,
		Default: defaultValue,
	})
}

func getHetznerClient(token string) *hcloud.Client {
	return hcloud.NewClient(hcloud.WithToken(token))
}

func formatServerType(st *hcloud.ServerType) string {
	return fmt.Sprintf("%s (CPU: %d cores, RAM: %.0f GB, Disk: %d GB)",
		st.Name,
		st.Cores,
		st.Memory,
		st.Disk)
}

func getHetznerOptions(client *hcloud.Client) (*HetznerOptions, error) {
	ctx := context.Background()
	options := &HetznerOptions{}

	// Get locations (regions)
	locations, _, err := client.Location.List(ctx, hcloud.LocationListOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get locations: %v", err)
	}
	options.Regions = make([]string, len(locations))
	for i, loc := range locations {
		options.Regions[i] = fmt.Sprintf("%s (%s)", loc.Name, loc.Description)
	}

	// Get server types
	serverTypes, _, err := client.ServerType.List(ctx, hcloud.ServerTypeListOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get server types: %v", err)
	}
	options.NodeTypes = make([]string, 0)
	for _, st := range serverTypes {
		if st.Architecture == "x86" {
			options.NodeTypes = append(options.NodeTypes, formatServerType(st))
		}
	}

	// TODO: Implement dynamic K8s version fetching
	options.K8sVersions = []string{"1.27", "1.26", "1.25"}

	return options, nil
}

func buildQuestions(options *HetznerOptions) []*survey.Question {
	return []*survey.Question{
		createInputQuestion("clusterName", "What is the name of your cluster?", "my-cluster"),
		createSelectQuestion("region", "Choose a region:", options.Regions),
		createInputQuestion("nodeCount", "How many master nodes do you want?", "3"),
		createInputQuestion("nodeCount", "How many worker nodes do you want?", "3"),
		createSelectQuestion("nodeType", "Choose node type:", options.NodeTypes),
		createSelectQuestion("kubernetesVersion", "Choose Kubernetes version:", options.K8sVersions),
	}
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	config := &ClusterConfig{}
	
	// Get API token
	tokenQ := createQuestion("hetznerToken", "Enter your Hetzner Cloud API token:", 
		&survey.Password{Message: "Enter your Hetzner Cloud API token:"})
	
	if err := survey.Ask([]*survey.Question{tokenQ}, config); err != nil {
		return fmt.Errorf("failed to get API token: %v", err)
	}

	// Initialize client and get options
	client := getHetznerClient(config.HetznerToken)
	options, err := getHetznerOptions(client)
	if err != nil {
		return fmt.Errorf("failed to retrieve options from Hetzner API: %v", err)
	}

	// Ask remaining questions
	if err := survey.Ask(buildQuestions(options), config); err != nil {
		return fmt.Errorf("failed to get cluster configuration: %v", err)
	}

	// TODO: Implement cluster creation logic
	return nil
}

func newBootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a new Kubernetes cluster",
		Long:  `Bootstrap will create a new Kubernetes cluster on Hetzner Cloud`,
		RunE:  runBootstrap,
	}
}

func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "hcloud-k8s",
		Short: "A CLI tool for managing Kubernetes clusters on Hetzner Cloud",
	}
	
	rootCmd.AddCommand(newBootstrapCmd())
	return rootCmd.Execute()
} 