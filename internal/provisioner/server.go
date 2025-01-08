package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// Config holds the server provisioning configuration
type Config struct {
	ClusterName       string
	Region           string
	NodeCount        int
	NodeType         string
	MasterNodes      []string
	WorkerNodes      []string
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

// ProvisionMasterNodes creates the master nodes for the cluster
func (p *ServerProvisioner) ProvisionMasterNodes() error {
	ctx := context.Background()
	
	// Extract base server type name (remove the specs part)
	serverType := strings.Split(p.config.NodeType, " ")[0]
	
	fmt.Printf("\nProvisioning %d master nodes...\n", p.config.NodeCount)
	for i := 1; i <= p.config.NodeCount; i++ {
		name := fmt.Sprintf("%s-master-%d", p.config.ClusterName, i)
		
		opts := hcloud.ServerCreateOpts{
			Name:       name,
			ServerType: &hcloud.ServerType{Name: serverType},
			Image:     &hcloud.Image{Name: "ubuntu-22.04"},
			Location:  &hcloud.Location{Name: strings.Split(p.config.Region, " ")[0]},
			Labels: map[string]string{
				"cluster":  p.config.ClusterName,
				"role":     "master",
				"index":    fmt.Sprintf("%d", i),
			},
		}

		result, _, err := p.client.Server.Create(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to create master node %s: %v", name, err)
		}

		fmt.Printf("Creating master node %s (ID: %d)...\n", name, result.Server.ID)
		if err := p.waitForServer(ctx, result.Action); err != nil {
			return fmt.Errorf("error waiting for master node %s creation: %v", name, err)
		}
		fmt.Println()

		p.config.MasterNodes = append(p.config.MasterNodes, fmt.Sprint(result.Server.ID))
	}

	fmt.Printf("\nAll master nodes provisioned successfully!\n")
	return nil
}

func (p *ServerProvisioner) waitForServer(ctx context.Context, action *hcloud.Action) error {
	done := make(chan error)
	
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
		}
	}
} 