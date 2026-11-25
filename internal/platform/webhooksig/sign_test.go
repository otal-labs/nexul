package webhooksig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerify_TakesOnlyTheSignatureOfThatBodyUnderThatSecret(t *testing.T) {
	sig := Sign("secret", []byte("payload"))

	assert.Equal(t, "sha256=b82fcb791acec57859b989b430a826488ce2e479fdf92326bd0a2e8375a42ba4", sig)
	assert.True(t, Verify("secret", []byte("payload"), sig))
	assert.False(t, Verify("other", []byte("payload"), sig))
	assert.False(t, Verify("secret", []byte("payload2"), sig))
	assert.False(t, Verify("secret", []byte("payload"), "sha256=deadbeef"))
}
