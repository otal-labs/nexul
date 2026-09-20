package dns

import (
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// RecordType is a DNS record class Nexul manages.
type RecordType string

const (
	RecordA     RecordType = "A"
	RecordAAAA  RecordType = "AAAA"
	RecordCNAME RecordType = "CNAME"
	RecordTXT   RecordType = "TXT"
)

// Zone is a DNS zone hosted at the provider.
type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

// Record is a DNS record as managed at the provider.
type Record struct {
	ID        string     `json:"id"`
	ZoneID    string     `json:"zone_id"`
	Type      RecordType `json:"type"`
	Name      string     `json:"name"`
	Content   string     `json:"content"`
	TTL       int        `json:"ttl"`
	Proxied   bool       `json:"proxied"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
}

// RecordInput is the validated input to a record create/update. Provider
// quirks (proxying, priorities) stay inside the provider package.
type RecordInput struct {
	Type    RecordType `json:"type"`
	Name    string     `json:"name"`
	Content string     `json:"content"`
	TTL     int        `json:"ttl"`
	// Proxied routes the record through Cloudflare; a CNAME to *.cfargotunnel.com resolves only when proxied.
	Proxied bool `json:"proxied"`
}

// Validate rejects records that cannot be created. TTL 0 means the provider's
// default; negative TTLs are always invalid.
func (r RecordInput) Validate() error {
	switch r.Type {
	case RecordA, RecordAAAA, RecordCNAME, RecordTXT:
	default:
		return fmt.Errorf("%w: unsupported record type %q", apperrs.ErrInvalid, r.Type)
	}
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("%w: record name is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(r.Content) == "" {
		return fmt.Errorf("%w: record content is required", apperrs.ErrInvalid)
	}
	if r.TTL < 0 {
		return fmt.Errorf("%w: record TTL cannot be negative", apperrs.ErrInvalid)
	}
	return nil
}

// ServiceHostname is the local association between a deployed service and its provider-created hostname record.
type ServiceHostname struct {
	Service   string     `json:"service"`
	Hostname  string     `json:"hostname"`
	ZoneID    string     `json:"zone_id"`
	Zone      string     `json:"zone"`
	RecordID  string     `json:"record_id"`
	Type      RecordType `json:"type"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
}

// ServiceHostnameInput is the validated input to Service.SetServiceHostname.
type ServiceHostnameInput struct {
	Service  string     `json:"service"`
	Hostname string     `json:"hostname"`
	ZoneID   string     `json:"zone_id"`
	Zone     string     `json:"zone"`
	Type     RecordType `json:"type"`
	Target   string     `json:"target"`
}

// Validate rejects hostname associations that cannot be created.
func (in ServiceHostnameInput) Validate() error {
	if strings.TrimSpace(in.Service) == "" {
		return fmt.Errorf("%w: service is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.Hostname) == "" {
		return fmt.Errorf("%w: hostname is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.ZoneID) == "" {
		return fmt.Errorf("%w: zone is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.Zone) == "" {
		return fmt.Errorf("%w: zone is required", apperrs.ErrInvalid)
	}
	switch in.Type {
	case RecordA, RecordAAAA, RecordCNAME, RecordTXT:
	default:
		return fmt.Errorf("%w: unsupported record type %q", apperrs.ErrInvalid, in.Type)
	}
	if strings.TrimSpace(in.Target) == "" {
		return fmt.Errorf("%w: target is required", apperrs.ErrInvalid)
	}
	return nil
}

// recordNameFor derives the record name for a hostname under a zone; not under it is an error, never guessed.
func recordNameFor(hostname, zone string) (string, error) {
	hostname = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), "."))
	zone = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(zone), "."))
	if zone == "" || hostname == "" {
		return "", fmt.Errorf("%w: hostname and zone are required", apperrs.ErrInvalid)
	}
	if hostname == zone {
		return "@", nil
	}
	if strings.HasSuffix(hostname, "."+zone) {
		return strings.TrimSuffix(hostname, "."+zone), nil
	}
	return "", fmt.Errorf(
		"%w: hostname %q is not under zone %q — the DNS record would not match the hostname",
		apperrs.ErrInvalid, hostname, zone)
}
