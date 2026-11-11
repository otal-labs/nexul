package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The 1Password SDK embeds a 9.5 MB wasm core; only nexul-server and nexul-runner carry it.
func TestNexulBinary_DoesNotLinkTheOnePasswordSDK(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "list", "-deps", ".").Output()
	require.NoError(t, err)
	for dep := range strings.SplitSeq(string(out), "\n") {
		require.NotContains(t, dep, "1password/onepassword-sdk-go", "cmd/nexul must not import the SDK")
		require.NotContains(t, dep, "platform/secretref/onepassword", "cmd/nexul must not import the 1Password resolver")
	}
}
