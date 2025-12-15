package provider

import (
	"testing"

	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/usgdns"
	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/webhook"
)

func TestNewProvider(t *testing.T) {
	client := usgdns.NewClient("http://test.local", "test-token")
	domainFilter := []string{"example.com"}

	provider := NewProvider(client, domainFilter, false)

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}

	if provider.client != client {
		t.Error("Expected client to be set")
	}

	if len(provider.domainFilter) != 1 || provider.domainFilter[0] != "example.com" {
		t.Error("Expected domain filter to be set")
	}

	if provider.dryRun != false {
		t.Error("Expected dry run to be false")
	}
}

func TestGetDomainFilter(t *testing.T) {
	client := usgdns.NewClient("http://test.local", "test-token")
	domainFilter := []string{"example.com", "test.local"}

	provider := NewProvider(client, domainFilter, false)
	filter := provider.GetDomainFilter()

	if len(filter.Filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(filter.Filters))
	}

	if filter.Filters[0] != "example.com" || filter.Filters[1] != "test.local" {
		t.Error("Expected filters to match")
	}
}

func TestNormalizeDNSName(t *testing.T) {
	client := usgdns.NewClient("http://test.local", "test-token")
	domainFilter := []string{"example.com"}
	provider := NewProvider(client, domainFilter, false)

	tests := []struct {
		input    string
		expected string
	}{
		{"test.example.com.", "test.example.com"},
		{"test.example.com", "test.example.com"},
		{"subdomain.test.example.com.", "subdomain.test.example.com"},
	}

	for _, tt := range tests {
		result := provider.normalizeDNSName(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeDNSName(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestAdjustEndpoints(t *testing.T) {
	client := usgdns.NewClient("http://test.local", "test-token")
	provider := NewProvider(client, []string{"example.com"}, false)

	endpoints := []*webhook.Endpoint{
		{
			DNSName:    "test.example.com",
			Targets:    []string{"1.2.3.4"},
			RecordType: "A",
		},
		{
			DNSName:    "test2.example.com",
			Targets:    []string{"example.com"},
			RecordType: "CNAME", // Should be filtered out
		},
		{
			DNSName: "test3.example.com",
			Targets: []string{"5.6.7.8"},
			// No RecordType, should default to A
		},
	}

	adjusted, err := provider.AdjustEndpoints(endpoints)
	if err != nil {
		t.Fatalf("AdjustEndpoints failed: %v", err)
	}

	// Should have 2 endpoints (A records only)
	if len(adjusted) != 2 {
		t.Errorf("Expected 2 adjusted endpoints, got %d", len(adjusted))
	}

	// Check that all are A records
	for _, ep := range adjusted {
		if ep.RecordType != "A" {
			t.Errorf("Expected RecordType A, got %s", ep.RecordType)
		}
		if ep.RecordTTL == 0 {
			t.Error("Expected RecordTTL to be set")
		}
	}
}
