package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"hcloud-k8s/internal/provisioner"
	"hcloud-k8s/internal/survey"
)

func runBootstrap(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Get cluster configuration (including token)
	config, err := survey.GetClusterConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster configuration: %v", err)
	}

	// Provision infrastructure
	fmt.Println("\nStarting cluster provisioning...")
	prov := provisioner.New(config.HetznerToken, &provisioner.Config{
		ClusterName: config.ClusterName,
		Region:     config.Region,
		NodeCount:  config.NodeCount,
		NodeType:   config.NodeType,
	})

	if err := prov.ProvisionMasterNodes(); err != nil {
		return fmt.Errorf("failed to provision cluster: %v", err)
	}

	fmt.Printf("\nCluster %s infrastructure provisioned successfully!\n", config.ClusterName)
	return nil
}

// BootstrapCmd creates and returns the cobra command for bootstrapping
func BootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a new Kubernetes cluster",
		Long:  `Bootstrap will create a new Kubernetes cluster on Hetzner Cloud`,
		RunE:  runBootstrap,
	}
}

// Execute adds all child commands to the root command
func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "hcloud-k8s",
		Short: "A CLI tool for managing Kubernetes clusters on Hetzner Cloud",
	}
	rootCmd.AddCommand(BootstrapCmd())
	return rootCmd.Execute()
} 