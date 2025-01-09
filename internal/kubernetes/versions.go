package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"hcloud-k8s/internal/errors"
)

const (
	userAgent = "hcloud-k8s/1.0"
	maxVersions = 5
)

// Distribution represents a Kubernetes distribution type (RKE2 or K3s)
type Distribution string

// Supported Kubernetes distributions
const (
	RKE2 Distribution = "RKE2"
	K3S  Distribution = "K3s"
)

// Release represents a GitHub release response structure
type Release struct {
	TagName string `json:"tag_name"`
}

type githubError struct {
	Message string `json:"message"`
}

// GetDistributions returns a list of supported Kubernetes distributions
func GetDistributions() []string {
	return []string{string(RKE2), string(K3S)}
}

// GetVersions fetches available versions for the specified Kubernetes distribution
// from GitHub releases. Returns the 5 most recent major.minor versions.
func GetVersions(ctx context.Context, dist Distribution) ([]string, error) {
	var url string
	switch dist {
	case RKE2:
		url = "https://api.github.com/repos/rancher/rke2/releases"
	case K3S:
		url = "https://api.github.com/repos/k3s-io/k3s/releases"
	default:
		return nil, errors.NewValidationError("distribution", fmt.Sprintf("unsupported distribution: %s", dist))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add required headers
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.NewAPIError("GitHub API", http.StatusServiceUnavailable, err.Error())
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		var githubErr githubError
		if err := json.NewDecoder(resp.Body).Decode(&githubErr); err != nil {
			return nil, errors.NewAPIError("GitHub API", resp.StatusCode, "failed to decode error response")
		}
		return nil, errors.NewAPIError("GitHub API", resp.StatusCode, githubErr.Message)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, errors.NewAPIError("GitHub API", http.StatusUnprocessableEntity, "failed to decode releases")
	}

	if len(releases) == 0 {
		return nil, errors.NewConfigError("Kubernetes", fmt.Sprintf("no releases found for %s", dist))
	}

	versions := make(map[string]struct{})
	for _, release := range releases {
		version := strings.TrimPrefix(release.TagName, "v")
		if strings.Count(version, ".") == 2 {
			majorMinor := strings.Join(strings.Split(version, ".")[:2], ".")
			versions[majorMinor] = struct{}{}
		}
	}

	if len(versions) == 0 {
		return nil, errors.NewConfigError("Kubernetes", fmt.Sprintf("no valid versions found for %s", dist))
	}

	uniqueVersions := make([]string, 0, len(versions))
	for v := range versions {
		uniqueVersions = append(uniqueVersions, v)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(uniqueVersions)))
	if len(uniqueVersions) > maxVersions {
		uniqueVersions = uniqueVersions[:maxVersions]
	}

	return uniqueVersions, nil
} 