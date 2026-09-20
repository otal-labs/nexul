//go:build !embed

package webui

import "testing"

func TestAssets_NoEmbed_ReturnsNil(t *testing.T) {
	if Assets() != nil {
		t.Fatal("Assets() without embed tag should be nil")
	}
}
