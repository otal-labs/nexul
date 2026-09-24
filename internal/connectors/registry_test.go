package connectors

import "testing"

func TestRegistryContainsExpectedEntries(t *testing.T) {
	want := []string{"github", "cloudflare"}
	all := Registry()

	got := make(map[string]Connector, len(all))
	for _, c := range all {
		got[c.ID] = c
	}

	for _, id := range want {
		c, ok := got[id]
		if !ok {
			t.Fatalf("registry missing expected connector %q", id)
		}
		if c.ID == "" {
			t.Errorf("connector %q: empty ID", id)
		}
		if c.Name == "" {
			t.Errorf("connector %q: empty Name", id)
		}
		if c.Description == "" {
			t.Errorf("connector %q: empty Description", id)
		}
		if c.OAuth != nil {
			t.Errorf("connector %q: expected OAuth == nil (no implementation wired yet), got %v", id, c.OAuth)
		}
	}
}

// TestLiveKitEntry_ManualCredential proves ticket 10's registry shape: a
// static Manual field list (no OAuth), the three fields the connector needs,
// with only the API secret marked Secret.
func TestLiveKitEntry_ManualCredential(t *testing.T) {
	byID := make(map[string]Connector)
	for _, c := range Registry() {
		byID[c.ID] = c
	}
	lk, ok := byID["livekit"]
	if !ok {
		t.Fatal("registry missing livekit connector")
	}
	if lk.OAuth != nil {
		t.Errorf("livekit.OAuth = %v, want nil (manual credential, not OAuth)", lk.OAuth)
	}
	if lk.Category != "communication" {
		t.Errorf("livekit.Category = %q, want communication", lk.Category)
	}
	wantKeys := map[string]bool{"ws_url": false, "api_key": false, "api_secret": true}
	if len(lk.Manual) != len(wantKeys) {
		t.Fatalf("livekit.Manual = %+v, want %d fields", lk.Manual, len(wantKeys))
	}
	for _, f := range lk.Manual {
		wantSecret, ok := wantKeys[f.Key]
		if !ok {
			t.Errorf("unexpected manual field %q", f.Key)
			continue
		}
		if f.Secret != wantSecret {
			t.Errorf("field %q Secret = %v, want %v", f.Key, f.Secret, wantSecret)
		}
		if f.Label == "" {
			t.Errorf("field %q has empty Label", f.Key)
		}
	}
}

// TestGitHubEntry_Description pins the card copy to capabilities gitprovider actually wires: PR and review
// linking, repository scanning, and push-driven deploys. There is no issue or PR creation write path.
func TestGitHubEntry_Description(t *testing.T) {
	for _, c := range Registry() {
		if c.ID != "github" {
			continue
		}
		want := "Links pull requests and reviews to tickets, scans your repositories, and deploys on push"
		if c.Description != want {
			t.Errorf("github.Description = %q, want %q", c.Description, want)
		}
		return
	}
	t.Fatal("registry missing github connector")
}

func TestGitProviderFlag(t *testing.T) {
	byID := make(map[string]Connector)
	var gitProviders []string
	for _, c := range Registry() {
		byID[c.ID] = c
		if c.GitProvider {
			gitProviders = append(gitProviders, c.ID)
		}
	}
	if want := []string{"github"}; len(gitProviders) != len(want) || gitProviders[0] != want[0] {
		t.Errorf("connectors with GitProvider == true = %v, want %v", gitProviders, want)
	}
	if byID["cloudflare"].GitProvider {
		t.Errorf("cloudflare.GitProvider = true, want false")
	}
}

func TestCloudflareEntry_OnlyAccessChecksAreAdvisory(t *testing.T) {
	advisory := map[string]bool{}
	for _, ch := range registry["cloudflare"].Checks {
		advisory[ch.Key] = ch.Advisory
	}
	want := map[string]bool{
		"token": false, "zone_read": false, "dns_edit": false, "tunnel_edit": false,
		"access_apps_edit": true, "access_tokens_edit": true,
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
