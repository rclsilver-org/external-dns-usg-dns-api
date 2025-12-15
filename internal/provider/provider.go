package provider

import (
	"fmt"
	"log"
	"strings"

	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/usgdns"
	"github.com/rclsilver-org/external-dns-usg-dns-api/internal/webhook"
)

// Provider implements the external-dns webhook provider for usg-dns-api
type Provider struct {
	client       *usgdns.Client
	domainFilter []string
	dryRun       bool
}

// NewProvider creates a new provider instance
func NewProvider(client *usgdns.Client, domainFilter []string, dryRun bool) *Provider {
	return &Provider{
		client:       client,
		domainFilter: domainFilter,
		dryRun:       dryRun,
	}
}

// GetDomainFilter returns the domain filter
func (p *Provider) GetDomainFilter() webhook.DomainFilter {
	return webhook.DomainFilter{
		Filters: p.domainFilter,
	}
}

// GetRecords returns all DNS records
func (p *Provider) GetRecords() ([]*webhook.Endpoint, error) {
	records, err := p.client.GetRecords()
	if err != nil {
		return nil, fmt.Errorf("failed to get records: %w", err)
	}

	endpoints := make([]*webhook.Endpoint, 0, len(records))
	for _, record := range records {
		// Only handle A records for now
		endpoints = append(endpoints, &webhook.Endpoint{
			DNSName:    record.Name,
			Targets:    []string{record.Target},
			RecordType: "A",
			RecordTTL:  300, // Default TTL
		})
	}

	return endpoints, nil
}

// ApplyChanges applies the given changes
func (p *Provider) ApplyChanges(changes *webhook.Changes) error {
	if p.dryRun {
		log.Println("[DRY RUN] Would apply changes:")
		log.Printf("[DRY RUN] Create: %d records", len(changes.Create))
		log.Printf("[DRY RUN] Update: %d records", len(changes.UpdateNew))
		log.Printf("[DRY RUN] Delete: %d records", len(changes.Delete))
		return nil
	}

	// Handle creates
	for _, endpoint := range changes.Create {
		if err := p.createRecord(endpoint); err != nil {
			return fmt.Errorf("failed to create record %s: %w", endpoint.DNSName, err)
		}
		log.Printf("Created record: %s -> %v", endpoint.DNSName, endpoint.Targets)
	}

	// Handle updates
	for i, oldEndpoint := range changes.UpdateOld {
		if i >= len(changes.UpdateNew) {
			break
		}
		newEndpoint := changes.UpdateNew[i]
		if err := p.updateRecord(oldEndpoint, newEndpoint); err != nil {
			return fmt.Errorf("failed to update record %s: %w", newEndpoint.DNSName, err)
		}
		log.Printf("Updated record: %s -> %v", newEndpoint.DNSName, newEndpoint.Targets)
	}

	// Handle deletes
	for _, endpoint := range changes.Delete {
		if err := p.deleteRecord(endpoint); err != nil {
			return fmt.Errorf("failed to delete record %s: %w", endpoint.DNSName, err)
		}
		log.Printf("Deleted record: %s", endpoint.DNSName)
	}

	return nil
}

// AdjustEndpoints adjusts endpoints (optional, can return as-is)
func (p *Provider) AdjustEndpoints(endpoints []*webhook.Endpoint) ([]*webhook.Endpoint, error) {
	// Filter out non-A records as usg-dns-api only handles A records
	adjusted := make([]*webhook.Endpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.RecordType == "A" || endpoint.RecordType == "" {
			// Ensure record type is set
			endpoint.RecordType = "A"
			// Set a default TTL if not set
			if endpoint.RecordTTL == 0 {
				endpoint.RecordTTL = 300
			}
			// Remove domain suffix if present in filter
			endpoint.DNSName = p.normalizeDNSName(endpoint.DNSName)
			adjusted = append(adjusted, endpoint)
		}
	}
	return adjusted, nil
}

func (p *Provider) normalizeDNSName(dnsName string) string {
	// Remove trailing dot if present
	dnsName = strings.TrimSuffix(dnsName, ".")

	// If domain filters are defined, try to make the name relative
	for _, filter := range p.domainFilter {
		filter = strings.TrimPrefix(filter, ".")
		if strings.HasSuffix(dnsName, "."+filter) {
			// Keep the full name for now - usg-dns-api might handle FQDNs
			break
		}
	}

	return dnsName
}

func (p *Provider) createRecord(endpoint *webhook.Endpoint) error {
	if len(endpoint.Targets) == 0 {
		return fmt.Errorf("no targets specified")
	}

	// Only support A records with a single target
	target := endpoint.Targets[0]

	_, err := p.client.CreateRecord(endpoint.DNSName, target)
	return err
}

func (p *Provider) updateRecord(oldEndpoint, newEndpoint *webhook.Endpoint) error {
	// First, find the record by name
	records, err := p.client.GetRecords()
	if err != nil {
		return fmt.Errorf("failed to get records: %w", err)
	}

	var recordID string
	for _, record := range records {
		if record.Name == oldEndpoint.DNSName {
			recordID = record.ID
			break
		}
	}

	if recordID == "" {
		return fmt.Errorf("record not found: %s", oldEndpoint.DNSName)
	}

	if len(newEndpoint.Targets) == 0 {
		return fmt.Errorf("no targets specified")
	}

	target := newEndpoint.Targets[0]
	_, err = p.client.UpdateRecord(recordID, newEndpoint.DNSName, target)
	return err
}

func (p *Provider) deleteRecord(endpoint *webhook.Endpoint) error {
	// Find the record by name
	records, err := p.client.GetRecords()
	if err != nil {
		return fmt.Errorf("failed to get records: %w", err)
	}

	var recordID string
	for _, record := range records {
		if record.Name == endpoint.DNSName {
			recordID = record.ID
			break
		}
	}

	if recordID == "" {
		// Record not found, consider it already deleted
		log.Printf("Record %s not found, considering it already deleted", endpoint.DNSName)
		return nil
	}

	return p.client.DeleteRecord(recordID)
}
