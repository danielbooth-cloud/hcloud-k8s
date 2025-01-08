package hetzner

import (
	"context"
	"fmt"
	"time"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const (
	// DefaultTimeout is the default timeout for API calls
	DefaultTimeout = 30 * time.Second
)

// Client wraps the Hetzner cloud client with our custom operations
type Client struct {
	*hcloud.Client
}

// New creates a new Hetzner client
func New(token string) *Client {
	return &Client{
		Client: hcloud.NewClient(hcloud.WithToken(token)),
	}
}

// GetLocations returns available Hetzner locations with descriptions
func (c *Client) GetLocations(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	locations, _, err := c.Location.List(ctx, hcloud.LocationListOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get locations: %v", err)
	}
	
	var names []string
	for _, loc := range locations {
		names = append(names, fmt.Sprintf("%s (%s)", loc.Name, loc.Description))
	}
	return names, nil
}

// GetServerTypes returns available server types with specifications
func (c *Client) GetServerTypes(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	serverTypes, _, err := c.ServerType.List(ctx, hcloud.ServerTypeListOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get server types: %v", err)
	}
	
	var types []string
	for _, st := range serverTypes {
		if st.Architecture == "x86" {
			types = append(types, fmt.Sprintf("%s (CPU: %d cores, RAM: %.0f GB, Disk: %d GB)",
				st.Name, st.Cores, st.Memory, st.Disk))
		}
	}
	
	if len(types) == 0 {
		return nil, fmt.Errorf("no x86 server types found")
	}
	
	return types, nil
} 