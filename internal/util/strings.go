package util

import (
	"fmt"
	"strings"
)

// ExtractFirstPart returns the first part of a string before a space
func ExtractFirstPart(s string) string {
	return strings.Split(s, " ")[0]
}

// ParseVersion removes 'v' prefix and returns major.minor version
func ParseVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[:2], ".")
	}
	return version
}

// FormatResourceName creates a consistent name for cluster resources
func FormatResourceName(clusterName, resourceType string, index int) string {
	return fmt.Sprintf("%s-%s-%d", clusterName, resourceType, index)
} 