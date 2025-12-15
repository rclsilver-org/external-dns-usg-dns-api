package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration
type Config struct {
	// USG DNS API configuration
	URL   string
	Token string

	// Domain filter
	DomainFilter []string

	// Server configuration
	Port       int
	HealthPort int

	// Options
	DryRun bool
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		URL:        os.Getenv("USG_DNS_URL"),
		Token:      os.Getenv("USG_DNS_TOKEN"),
		Port:       8888, // Default port as per external-dns spec
		HealthPort: 8080, // Default health port
		DryRun:     false,
	}

	// Validate required fields
	if config.URL == "" {
		return nil, fmt.Errorf("USG_DNS_URL environment variable is required")
	}

	if config.Token == "" {
		return nil, fmt.Errorf("USG_DNS_TOKEN environment variable is required")
	}

	// Parse domain filter
	domainFilterStr := os.Getenv("DOMAIN_FILTER")
	if domainFilterStr != "" {
		config.DomainFilter = strings.Split(domainFilterStr, ",")
		// Trim spaces from each filter
		for i := range config.DomainFilter {
			config.DomainFilter[i] = strings.TrimSpace(config.DomainFilter[i])
		}
	}

	// Parse port
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
		}
		config.Port = port
	}

	// Parse health port
	if healthPortStr := os.Getenv("HEALTH_PORT"); healthPortStr != "" {
		healthPort, err := strconv.Atoi(healthPortStr)
		if err != nil {
			return nil, fmt.Errorf("invalid HEALTH_PORT: %w", err)
		}
		config.HealthPort = healthPort
	}

	// Parse dry run
	if dryRunStr := os.Getenv("DRY_RUN"); dryRunStr != "" {
		dryRun, err := strconv.ParseBool(dryRunStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DRY_RUN: %w", err)
		}
		config.DryRun = dryRun
	}

	return config, nil
}
