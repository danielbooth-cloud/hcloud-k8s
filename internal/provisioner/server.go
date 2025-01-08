package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"hcloud-k8s/internal/ctxutil"
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
func (p *ServerProvisioner) ProvisionCluster(ctx context.Context) error {
	// Create context with timeout for the entire provisioning operation
	ctx, cancel := ctxutil.WithTimeout(ctx, ctxutil.LongTimeout)
	defer cancel()

	if err := p.provisionNodes(ctx, Master, p.config.MasterNodeCount); err != nil {
		return fmt.Errorf("failed to provision master nodes: %v", err)
	}

	if err := p.provisionNodes(ctx, Worker, p.config.WorkerNodeCount); err != nil {
		return fmt.Errorf("failed to provision worker nodes: %v", err)
	}

	return nil
}

// provisionNodes creates nodes of the specified type
func (p *ServerProvisioner) provisionNodes(ctx context.Context, nodeType NodeType, count int) error {
	for i := 1; i <= count; i++ {
		// Create context with timeout for each server creation
		serverCtx, cancel := ctxutil.WithTimeout(ctx, ctxutil.DefaultTimeout)
		defer cancel()

		name := fmt.Sprintf("%s-%s-%d", p.config.ClusterName, nodeType, i)
		if err := p.createServer(serverCtx, name, nodeType); err != nil {
			return fmt.Errorf("failed to create %s node %s: %v", nodeType, name, err)
		}
	}
	return nil
}

func (p *ServerProvisioner) createServer(ctx context.Context, name string, nodeType NodeType) error {
	serverType := strings.Split(p.config.NodeType, " ")[0]
	
	opts := hcloud.ServerCreateOpts{
		Name:       name,
		ServerType: &hcloud.ServerType{Name: serverType},
		Image:     &hcloud.Image{Name: "ubuntu-22.04"},
		Location:  &hcloud.Location{Name: strings.Split(p.config.Region, " ")[0]},
		Labels: map[string]string{
			"cluster": p.config.ClusterName,
			"role":    string(nodeType),
			"index":   strings.Split(name, "-")[2],
		},
	}

	result, _, err := p.client.Server.Create(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to create server %s: %v", name, err)
	}

	fmt.Printf("Creating %s node %s (ID: %d)...\n", nodeType, name, result.Server.ID)
	if err := p.waitForServer(ctx, result.Action); err != nil {
		return err
	}
	fmt.Println()

	// Store the node ID in the appropriate slice
	if nodeType == Master {
		p.config.MasterNodes = append(p.config.MasterNodes, fmt.Sprint(result.Server.ID))
	} else {
		p.config.WorkerNodes = append(p.config.WorkerNodes, fmt.Sprint(result.Server.ID))
	}

	return nil
}

func (p *ServerProvisioner) waitForServer(ctx context.Context, action *hcloud.Action) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Creating server..."
	s.Start()
	defer s.Stop()

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
				
				s.Suffix = fmt.Sprintf(" Creating server... %d%%", action.Progress)
				
			case <-ctx.Done():
				done <- ctx.Err()
				return
			}
		}
	}()

	return <-done
} 