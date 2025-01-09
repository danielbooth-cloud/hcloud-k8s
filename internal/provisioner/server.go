package provisioner

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"hcloud-k8s/internal/util"
	"hcloud-k8s/internal/errors"
	"hcloud-k8s/internal/logging"
	"github.com/briandowns/spinner"
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
	logger *slog.Logger
}

// New creates a new ServerProvisioner instance
func New(token string, config *Config) *ServerProvisioner {
	return &ServerProvisioner{
		client: hcloud.NewClient(hcloud.WithToken(token)),
		config: config,
		logger: logging.GetLogger("provisioner"),
	}
}

// ProvisionCluster creates both master and worker nodes
func (p *ServerProvisioner) ProvisionCluster(ctx context.Context) error {
	p.logger.Info("starting cluster provisioning")

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
	p.logger.Info("starting node provisioning", 
		"nodeType", nodeType,
		"count", count,
		"cluster", p.config.ClusterName)

	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s-%s-%d", p.config.ClusterName, nodeType, i)
		
		// Use the parent context directly since it already has a timeout
		if err := p.createServer(ctx, name, nodeType); err != nil {
			p.logger.Error("failed to create node",
				"nodeType", nodeType,
				"name", name,
				"error", err)
			return fmt.Errorf("failed to create %s node %s: %v", nodeType, name, err)
		}
		
		p.logger.Info("successfully created node",
			"nodeType", nodeType,
			"name", name)
	}
	return nil
}

func (p *ServerProvisioner) createServer(ctx context.Context, name string, nodeType NodeType) error {
	// Generate or retrieve the SSH key
	sshKey, err := p.generateSSHKey(ctx, p.config.ClusterName)
	if err != nil {
		return err
	}

	serverType := util.ExtractFirstPart(p.config.NodeType)
	location := util.ExtractFirstPart(p.config.Region)

	opts := hcloud.ServerCreateOpts{
		Name:       name,
		ServerType: &hcloud.ServerType{Name: serverType},
		Image:      &hcloud.Image{Name: "ubuntu-22.04"},
		Location:   &hcloud.Location{Name: location},
		SSHKeys:    []*hcloud.SSHKey{sshKey},
		Labels: map[string]string{
			"cluster": p.config.ClusterName,
			"role":    string(nodeType),
			"index":   strings.Split(name, "-")[2],
		},
	}

	result, _, err := p.client.Server.Create(ctx, opts)
	if err != nil {
		return errors.NewAPIError("CreateServer", http.StatusInternalServerError, err.Error())
	}

	return p.waitForServer(ctx, result.Action)
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