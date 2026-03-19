package cmd

import (
	"context"
	"fmt"
)

// init a struct for a single item
type Cluster struct {
	Name string
	Path string
}

// init a grouping struct
type Clusters struct {
	Cluster []Cluster
}

// Config holds application-wide configuration
type Config struct {
	// Directory configuration
	BaseDir      string
	ClusterDir   string
	ComponentDir string

	// Output configuration
	ColorOutput bool

	// Jsonnet configuration
	JsonnetPaths []string
	ExtVarFiles  []string
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		BaseDir:      ".",
		ColorOutput:  true,
		JsonnetPaths: []string{},
		ExtVarFiles:  []string{},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.BaseDir == "" {
		return fmt.Errorf("base directory cannot be empty")
	}
	return nil
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const configContextKey contextKey = "kr8-config"

// SetConfigInContext stores the Config in a context
func SetConfigInContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configContextKey, cfg)
}

// GetConfigFromContext retrieves the Config from a context
func GetConfigFromContext(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(configContextKey).(*Config); ok {
		return cfg
	}
	// Return default config if not found (shouldn't happen in normal operation)
	return NewConfig()
}
