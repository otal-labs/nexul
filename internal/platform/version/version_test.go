package version

import "testing"

func TestChannel(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"dev build", "dev", "dev"},
		{"beta build", "v0.2.0-beta-003", "beta"},
		{"stable build", "v0.2.0", "stable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := Version
			t.Cleanup(func() { Version = orig })
			Version = tt.version

			if got := Channel(); got != tt.want {
				t.Fatalf("Channel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsRelease(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"dev build", "dev", false},
		{"beta build", "v0.2.0-beta-003", true},
		{"stable build", "v0.2.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := Version
			t.Cleanup(func() { Version = orig })
			Version = tt.version

			if got := IsRelease(); got != tt.want {
				t.Fatalf("IsRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewer(t *testing.T) {
	tests := []struct {
		name               string
		release, installed string
		want               bool
	}{
		{"a later beta", "v0.3.41-beta", "v0.3.40-beta", true},
		{"an installed beta that is ahead", "v0.3.40-beta", "v0.3.41-beta", false},
		{"the same release", "v0.3.40-beta", "v0.3.40-beta", false},
		{"a later numbered beta", "v0.2.0-beta-123", "v0.2.0-beta-122", true},
		{"an installed numbered beta that is ahead", "v0.2.0-beta-122", "v0.2.0-beta-123", false},
		{"a later dotted beta", "v0.2.0-beta.10", "v0.2.0-beta.9", true},
		{"a stable release after its beta", "v0.2.0", "v0.2.0-beta.9", true},
		{"a dev build takes any release", "v0.2.0", "dev", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Newer(tt.release, tt.installed); got != tt.want {
				t.Fatalf("Newer(%q, %q) = %v, want %v", tt.release, tt.installed, got, tt.want)
			}
		})
	}
}
