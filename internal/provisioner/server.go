package provisioner

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"hcloud-k8s/internal/errors"
	"hcloud-k8s/internal/logging"
	"github.com/briandowns/spinner"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"hcloud-k8s/internal/util"
	"os/exec"
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

func (p *ServerProvisioner) getLatestUbuntuImage(ctx context.Context) (*hcloud.Image, error) {
	images, err := p.client.Image.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get images: %v", err)
	}

	latestImage := p.findLatestImageByFilter(images, func(img *hcloud.Image) bool {
		return strings.Contains(strings.ToLower(img.Name), "ubuntu") && 
			   img.Architecture == "x86"
	})

	if latestImage == nil {
		return nil, fmt.Errorf("no x86 Ubuntu images found")
	}

	p.logger.Info("selected Ubuntu image", 
		"name", latestImage.Name,
		"architecture", latestImage.Architecture,
		"created", latestImage.Created)

	return latestImage, nil
}

// findLatestImageByFilter returns the most recently created image that matches the filter criteria
func (p *ServerProvisioner) findLatestImageByFilter(images []*hcloud.Image, filter func(*hcloud.Image) bool) *hcloud.Image {
	var latest *hcloud.Image
	
	for _, img := range images {
		if !filter(img) {
			continue
		}
		
		if latest == nil || img.Created.After(latest.Created) {
			latest = img
		}
	}
	
	return latest
}

func (p *ServerProvisioner) createServer(ctx context.Context, name string, nodeType NodeType) error {
	sshKey, err := p.generateSSHKey(ctx, p.config.ClusterName)
	if err != nil {
		return err
	}

	serverType := util.ExtractFirstPart(p.config.NodeType)
	location := util.ExtractFirstPart(p.config.Region)

	image, err := p.getLatestUbuntuImage(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest Ubuntu image: %v", err)
	}

	// Determine if this is the first master node
	isFirstMaster := nodeType == Master && strings.HasSuffix(name, "-1")

	// Create user data for first master node
	var userData string
	if isFirstMaster {
		userData = `#cloud-config
package_update: true
package_upgrade: true

packages:
  - nfs-common

write_files:
  - path: /etc/systemd/resolved.conf
    content: |
      [Resolve]
      DNS=1.1.1.1
      FallbackDNS=1.0.0.1

runcmd:
  - systemctl restart systemd-resolved
  - curl -sfL https://get.rke2.io | INSTALL_RKE2_TYPE="agent" sh -
  - systemctl enable rke2-server.service
  - systemctl start rke2-server.service`
	}

	opts := hcloud.ServerCreateOpts{
		Name:       name,
		ServerType: &hcloud.ServerType{Name: serverType},
		Image:      image,
		Location:   &hcloud.Location{Name: location},
		SSHKeys:    []*hcloud.SSHKey{sshKey},
		UserData:   userData,
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

	if err := p.waitForServer(ctx, result.Action); err != nil {
		return err
	}

	// If this is the first master node, wait for RKE2 server to be ready
	if isFirstMaster {
		server, _, err := p.client.Server.GetByID(ctx, result.Server.ID)
		if err != nil {
			return fmt.Errorf("failed to get server details: %v", err)
		}

		if err := p.waitForRKE2Server(ctx, server.PublicNet.IPv4.IP.String()); err != nil {
			return fmt.Errorf("failed waiting for RKE2 server: %v", err)
		}
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

func (p *ServerProvisioner) waitForRKE2Server(ctx context.Context, serverIP string) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Waiting for RKE2 server to be ready..."
	s.Start()
	defer s.Stop()

	// Maximum wait time of 5 minutes
	timeout := time.After(5 * time.Minute)
	tick := time.Tick(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for RKE2 server to be ready")
		case <-tick:
			// Check if RKE2 server is running using SSH
			cmd := exec.Command("ssh", 
				"-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null",
				"-o", "LogLevel=ERROR", // Suppress SSH warnings
				"-i", fmt.Sprintf("%s-key_id_rsa", p.config.ClusterName),
				"root@"+serverIP,
				"systemctl is-active rke2-server.service")

			output, err := cmd.CombinedOutput()
			status := strings.TrimSpace(string(output))
			
			if err == nil && status == "active" {
				p.logger.Info("RKE2 server is ready",
					"status", "active",
					"server", serverIP)
				return nil
			}
			
			s.Suffix = " Waiting for RKE2 server to be ready..."
		}
	}
} 