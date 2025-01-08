package ctxutil

import (
	"context"
	"time"
)

const (
	// DefaultTimeout is used for regular API calls
	DefaultTimeout = 30 * time.Second
	// LongTimeout is used for operations that take longer, like server provisioning
	LongTimeout = 10 * time.Minute
)

// NewLongTimeout creates a new context with the LongTimeout duration
func NewLongTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), LongTimeout)
} 