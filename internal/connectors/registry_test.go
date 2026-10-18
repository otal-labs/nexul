package connectors

import (
	"slices"
	"strings"
	"testing"
)

func TestRegistry_ListsEveryConnectorSortedByIDWithTheFieldsACardNeeds(t *testing.T) {
	all := Registry()
	if len(all) != len(registry) {
		t.Fatalf("Registry() lists %d connectors, the registry holds %d", len(all), len(registry))
	}
	if !slices.IsSortedFunc(all, func(a, b Connector) int { return strings.Compare(a.ID, b.ID) }) {
		t.Errorf("Registry() is not sorted by ID")
	}
	for _, c := range all {
		if c.ID == "" || c.Name == "" || c.Description == "" {
			t.Errorf("connector %+v is missing an ID, Name, or Description", c)
		}
	}
}

func TestCloudflareEntry_OnlyTheOptionalChecksAreAdvisory(t *testing.T) {
	advisory := map[string]bool{}
	for _, ch := range registry["cloudflare"].Checks {
		advisory[ch.Key] = ch.Advisory
	}
	want := map[string]bool{
		"token": false, "zone_read": false, "dns_edit": false, "tunnel_edit": false,
		"access_apps_edit": true, "access_tokens_edit": true, "analytics_read": true,
	}
	for key, wantAdvisory := range want {
		got, ok := advisory[key]
		if !ok {
			t.Errorf("cloudflare is missing check %q", key)
			continue
		}
		if got != wantAdvisory {
			t.Errorf("cloudflare check %q advisory = %v, want %v", key, got, wantAdvisory)
		}
	}
}
