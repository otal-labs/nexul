package license

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools declares the instance settings tool; the license read rides on instance_get.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{instanceUpdateTool(s)}
}

type instanceUpdateIn struct {
	LicenseKey *string `json:"license_key,omitempty" jsonschema:"A license key's text, replacing the current key; an empty string removes it."`
}

// instanceSettings is instance_update's answer, keyed so later instance settings sit beside the license.
type instanceSettings struct {
	License Status `json:"license"`
}

func instanceUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("instance_update", "Update instance",
		"Changes the instance's own settings; today, its enterprise license key. A license_key with a key's text replaces the "+
			"current key, refused with the reason when the key names another address, fails its signature or has expired. An "+
			"empty license_key removes it: the enterprise features stop and what they made stays. Returns the license as "+
			"instance_get shows it.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in instanceUpdateIn) (any, error) {
			if in.LicenseKey == nil {
				st, err := s.Status(ctx)
				return instanceSettings{License: st}, err
			}
			if *in.LicenseKey == "" {
				st, err := s.Remove(ctx)
				return instanceSettings{License: st}, err
			}
			st, err := s.Set(ctx, *in.LicenseKey)
			return instanceSettings{License: st}, err
		})
}
