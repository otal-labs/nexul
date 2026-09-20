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
