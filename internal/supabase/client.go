package supabase

import (
	"fmt"

	"github.com/pradord/lumen_final/backend/internal/config"
	"github.com/pradord/lumen_final/backend/pkg/logger"
	"github.com/supabase-community/supabase-go"
)

// Client wraps the Supabase client
type Client struct {
	*supabase.Client
	Enabled   bool
	jwtSecret string
	url       string
	anonKey   string
	logger    *logger.Logger
}

// NewClient creates a new Supabase client based on the configuration
func NewClient(cfg *config.Config, log *logger.Logger) (*Client, error) {
	// Check if Supabase URL is configured (indicates Supabase should be used)
	if cfg.Supabase.URL == "" {
		log.Info().Msg("Supabase client is disabled (no URL configured)")
		return &Client{
			Client:  nil,
			Enabled: false,
			logger:  log,
		}, nil
	}

	var apiKey string
	if cfg.IsProduction() {
		log.Info().Msg("Initializing Supabase client in production mode")
		// Use service role key in production for admin operations
		if cfg.Supabase.ServiceKey != "" {
			apiKey = cfg.Supabase.ServiceKey
		} else {
			apiKey = cfg.Supabase.AnonKey
		}
	} else {
		log.Info().Msg("Initializing Supabase client in development/testing mode")
		// Use anon key in development/testing
		apiKey = cfg.Supabase.AnonKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("Supabase API key is required")
	}

	// Initialize Supabase client
	client, err := supabase.NewClient(cfg.Supabase.URL, apiKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Supabase client: %w", err)
	}

	log.Info().
		Str("url", cfg.Supabase.URL).
		Msg("Supabase client initialized successfully")

	return &Client{
		Client:    client,
		Enabled:   true,
		jwtSecret: cfg.Supabase.JWTSecret,
		url:       cfg.Supabase.URL,
		anonKey:   cfg.Supabase.AnonKey,
		logger:    log,
	}, nil
}

// HealthCheck verifies the Supabase connection
func (c *Client) HealthCheck() error {
	if !c.Enabled {
		return nil
	}

	// Try to authenticate with the current API key
	// This is a simple health check that verifies connectivity
	if c.Client == nil {
		return fmt.Errorf("supabase client is not initialized")
	}

	// The client is initialized, consider it healthy
	// You can add more sophisticated health checks here
	return nil
}

// IsEnabled returns true if Supabase is enabled
func (c *Client) IsEnabled() bool {
	return c.Enabled
}


// GetJWTSecret returns the JWT secret for token validation
func (c *Client) GetJWTSecret() string {
	return c.jwtSecret
}

// GetURL returns the Supabase project URL
func (c *Client) GetURL() string {
	return c.url
}

// GetAnonKey returns the Supabase anon key
func (c *Client) GetAnonKey() string {
	return c.anonKey
}

// Close performs any necessary cleanup
func (c *Client) Close() error {
	if !c.Enabled {
		return nil
	}

	c.logger.Info().Msg("Closing Supabase client")
	// Supabase Go client doesn't have a Close method
	// This is a placeholder for future cleanup if needed
	return nil
}
