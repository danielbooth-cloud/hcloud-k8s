package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"hcloud-k8s/internal/ctxutil"
	"hcloud-k8s/internal/provisioner"
	"hcloud-k8s/internal/survey"
	"hcloud-k8s/internal/logging"
)

func runBootstrap(cmd *cobra.Command, args []string) error {
	logger := logging.GetLogger("bootstrap")
	
	ctx, cancel := ctxutil.NewLongTimeout()
	defer cancel()
	
	logger.Info("starting cluster bootstrap")
	
	config, err := survey.GetClusterConfig(ctx)
	if err != nil {
		logger.Error("failed to get cluster configuration", "error", err)
		return fmt.Errorf("failed to get cluster configuration: %v", err)
	}

	logger.Info("provisioning infrastructure",
		"cluster", config.ClusterName,
		"region", config.Region,
		"masterNodes", config.MasterNodeCount,
		"workerNodes", config.WorkerNodeCount)

	prov := provisioner.New(config.HetznerToken, &provisioner.Config{
		ClusterName:     config.ClusterName,
		Region:         config.Region,
		MasterNodeCount: config.MasterNodeCount,
		WorkerNodeCount: config.WorkerNodeCount,
		NodeType:       config.NodeType,
	})

	if err := prov.ProvisionCluster(ctx); err != nil {
		logger.Error("failed to provision cluster", 
			"cluster", config.ClusterName,
			"error", err)
		return fmt.Errorf("failed to provision cluster: %v", err)
	}

	logger.Info("cluster provisioned successfully",
		"cluster", config.ClusterName)
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