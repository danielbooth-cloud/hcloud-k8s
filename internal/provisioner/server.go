package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// NodeType represents the type of node (master or worker)
type NodeType string

// NodeType represents the type of node (master or worker)
const (
	Master NodeType = "master"
	Worker NodeType = "worker"
)

// Config holds the server provisioning configuration
type Config struct {
	ClusterName      string
	Region          string
	MasterNodeCount int
	WorkerNodeCount int
	NodeType        string
	MasterNodes     []string
	WorkerNodes     []string
}

// ServerProvisioner handles the creation of servers in Hetzner Cloud
type ServerProvisioner struct {
	client *hcloud.Client
	config *Config
}

// New creates a new ServerProvisioner instance
func New(token string, config *Config) *ServerProvisioner {
	return &ServerProvisioner{
		client: hcloud.NewClient(hcloud.WithToken(token)),
		config: config,
	}
}

// ProvisionCluster creates both master and worker nodes
func (p *ServerProvisioner) ProvisionCluster() error {
	if err := p.provisionNodes(Master, p.config.MasterNodeCount); err != nil {
		return fmt.Errorf("failed to provision master nodes: %v", err)
	}

	if err := p.provisionNodes(Worker, p.config.WorkerNodeCount); err != nil {
		return fmt.Errorf("failed to provision worker nodes: %v", err)
	}

	return nil
}

// provisionNodes creates nodes of the specified type
func (p *ServerProvisioner) provisionNodes(nodeType NodeType, count int) error {
	ctx := context.Background()
	serverType := strings.Split(p.config.NodeType, " ")[0]
	
	fmt.Printf("\nProvisioning %d %s nodes...\n", count, nodeType)
	
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s-%s-%d", p.config.ClusterName, nodeType, i)
		
		opts := hcloud.ServerCreateOpts{
			Name:       name,
			ServerType: &hcloud.ServerType{Name: serverType},
			Image:     &hcloud.Image{Name: "ubuntu-22.04"},
			Location:  &hcloud.Location{Name: strings.Split(p.config.Region, " ")[0]},
			Labels: map[string]string{
				"cluster": p.config.ClusterName,
				"role":    string(nodeType),
				"index":   fmt.Sprintf("%d", i),
			},
		}

		result, _, err := p.client.Server.Create(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to create %s node %s: %v", nodeType, name, err)
		}

		fmt.Printf("Creating %s node %s (ID: %d)...\n", nodeType, name, result.Server.ID)
		if err := p.waitForServer(ctx, result.Action); err != nil {
			return fmt.Errorf("error waiting for %s node %s creation: %v", nodeType, name, err)
		}
		fmt.Println()

		// Store the node ID in the appropriate slice
		if nodeType == Master {
			p.config.MasterNodes = append(p.config.MasterNodes, fmt.Sprint(result.Server.ID))
		} else {
			p.config.WorkerNodes = append(p.config.WorkerNodes, fmt.Sprint(result.Server.ID))
		}
	}

	fmt.Printf("\nAll %s nodes provisioned successfully!\n", nodeType)
	return nil
}

func (p *ServerProvisioner) waitForServer(ctx context.Context, action *hcloud.Action) error {
	done := make(chan error)
	progress := make(chan int)
	
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				action, _, err := p.client.Action.GetByID(ctx, action.ID)
				if err != nil {
					done <- err
					return
				}
				
				if action.Status == hcloud.ActionStatusSuccess {
					done <- nil
					return
				}
				
				if action.Status == hcloud.ActionStatusError {
					done <- fmt.Errorf("server creation failed: %s", action.ErrorMessage)
					return
				}
				
				progress <- action.Progress
				
			case <-ctx.Done():
				done <- ctx.Err()
				return
			}
		}
	}()

	for {
		select {
		case err := <-done:
			return err
		case prog := <-progress:
			fmt.Printf("\rProgress: %d%%", prog)
		}
	}
} 