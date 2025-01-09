package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type Distribution string

const (
	RKE2 Distribution = "RKE2"
	K3S  Distribution = "K3s"
)

type Release struct {
	TagName string `json:"tag_name"`
}

func GetDistributions() []string {
	return []string{string(RKE2), string(K3S)}
}

func GetVersions(ctx context.Context, dist Distribution) ([]string, error) {
	var url string
	switch dist {
	case RKE2:
		url = "https://api.github.com/repos/rancher/rke2/releases"
	case K3S:
		url = "https://api.github.com/repos/k3s-io/k3s/releases"
	default:
		return nil, fmt.Errorf("unsupported distribution: %s", dist)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	versions := make(map[string]struct{})
	for _, release := range releases {
		version := strings.TrimPrefix(release.TagName, "v")
		if strings.Count(version, ".") == 2 {
			majorMinor := strings.Join(strings.Split(version, ".")[:2], ".")
			versions[majorMinor] = struct{}{}
		}
	}

	uniqueVersions := make([]string, 0, len(versions))
	for v := range versions {
		uniqueVersions = append(uniqueVersions, v)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(uniqueVersions)))
	if len(uniqueVersions) > 5 {
		uniqueVersions = uniqueVersions[:5]
	}

	return uniqueVersions, nil
} 