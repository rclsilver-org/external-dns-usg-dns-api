package main

import (
	"log"
	"os"

	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/config"
	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/provider"
	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/server"
	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/usgdns"
	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/version"
)

func main() {
	log.Printf("Starting external-dns-usg-dns-api %s", version.VersionFull())

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	log.Printf("Configuration loaded:")
	log.Printf("  USG DNS URL: %s", cfg.URL)
	log.Printf("  Domain Filter: %v", cfg.DomainFilter)
	log.Printf("  API Port: %d", cfg.Port)
	log.Printf("  Health Port: %d", cfg.HealthPort)
	log.Printf("  Dry Run: %v", cfg.DryRun)

	// Create USG DNS API client
	client := usgdns.NewClient(cfg.URL, cfg.Token)

	// Create provider
	prov := provider.NewProvider(client, cfg.DomainFilter, cfg.DryRun)

	// Create and start server
	srv := server.NewServer(prov, cfg.Port, cfg.HealthPort)

	log.Printf("API server listening on port %d", cfg.Port)
	log.Printf("Health server listening on port %d", cfg.HealthPort)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
		os.Exit(1)
	}
}
