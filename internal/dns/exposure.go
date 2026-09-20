package dns

import (
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Exposure is a hostname routed through a gateway to a container; the provider record (CNAME or A/AAAA) lives there.
type Exposure struct {
	ID        string `json:"id"`
	GatewayID string `json:"gateway_id"`
	Hostname  string `json:"hostname"`
	// ServiceID is the container this exposure routes to (services.id); the primary key going forward.
	ServiceID string `json:"service_id,omitempty"`
	// Service is the legacy name column: the container's runtime name for a wizard-created exposure, or the
	// stack name for one created through the branch-deploy flow, which still keys by it.
	Service   string    `json:"service,omitempty"`
	Port      int       `json:"port"`
	ZoneID    string    `json:"zone_id"`
	Zone      string    `json:"zone"`
	RecordID  string    `json:"record_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateExposureInput is the validated input to Service.CreateExposure.
type CreateExposureInput struct {
	// GatewayID is optional: when empty, CreateExposure reuses a gateway on the target's machine (preferring
	// one already on one of its networks) or provisions one (spec §7).
	GatewayID string `json:"gateway_id,omitempty"`
	Hostname  string `json:"hostname"`
	// ServiceID targets a container directly (services.id). Service is the deprecated fallback: a bare name
	// resolved to a stack, whose single container becomes the target.
	ServiceID string `json:"service_id,omitempty"`
	Service   string `json:"service,omitempty"`
	Port      int    `json:"port"`
	// ZoneID/Zone are the exposure's own DNS zone; used to create its record and, when no gateway is reused,
	// to provision a new one.
	ZoneID string `json:"zone_id"`
	Zone   string `json:"zone"`
	// Kind picks the gateway kind when one must be provisioned; empty defaults to tunnel when Cloudflare is
	// connected, else proxy (spec §7).
	Kind GatewayKind `json:"kind,omitempty"`
}

// Validate rejects exposures that cannot be created.
func (in CreateExposureInput) Validate() error {
	if strings.TrimSpace(in.Hostname) == "" {
		return fmt.Errorf("%w: hostname is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.ServiceID) == "" && strings.TrimSpace(in.Service) == "" {
		return fmt.Errorf("%w: service_id (or the legacy service name) is required", apperrs.ErrInvalid)
	}
	if in.Port <= 0 {
		return fmt.Errorf("%w: container port must be positive", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.ZoneID) == "" || strings.TrimSpace(in.Zone) == "" {
		return fmt.Errorf("%w: zone is required", apperrs.ErrInvalid)
	}
	if in.Kind != "" {
		switch in.Kind {
		case GatewayTunnel, GatewayProxy:
		default:
			return fmt.Errorf("%w: unsupported gateway kind %q", apperrs.ErrInvalid, in.Kind)
		}
	}
	return nil
}

// ExposureSummary is the token-free view deploy needs to inject routing labels: hostname and port only.
type ExposureSummary struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
}

// tunnelOrigin builds the ingress URL cloudflared dials, via Docker's embedded DNS resolving the container name.
func tunnelOrigin(containerName string, port int) string {
	return fmt.Sprintf("http://%s:%d", containerName, port)
}
