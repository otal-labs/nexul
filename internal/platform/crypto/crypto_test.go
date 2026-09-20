package crypto

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := DeriveKey("dev-secret")
	got, err := Encrypt(key, []byte("cf-access-token"))
	require.NoError(t, err)
	assert.NotContains(t, got, "cf-access-token", "ciphertext must not leak the plaintext")

	dec, err := Decrypt(key, got)
	require.NoError(t, err)
	assert.Equal(t, "cf-access-token", string(dec))
}

func TestEncrypt_ProducesFreshCiphertext(t *testing.T) {
	key := DeriveKey("dev-secret")
	a, err := Encrypt(key, []byte("same-token"))
	require.NoError(t, err)
	b, err := Encrypt(key, []byte("same-token"))
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "the same token must not encrypt to the same value twice")
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	enc, err := Encrypt(DeriveKey("secret-a"), []byte("token"))
	require.NoError(t, err)
	_, err = Decrypt(DeriveKey("secret-b"), enc)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDecrypt))
}

func TestDecrypt_TamperedCiphertextFails(t *testing.T) {
	key := DeriveKey("dev-secret")
	enc, err := Encrypt(key, []byte("token"))
	require.NoError(t, err)
	b := []byte(enc)
	b[len(b)/2] ^= 0x01
	_, err = Decrypt(key, string(b))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDecrypt))
}

func TestDecrypt_GarbageInputFails(t *testing.T) {
	_, err := Decrypt(DeriveKey("dev-secret"), "!!!not-base64!!!")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDecrypt))
}

func TestDecrypt_EmptyInputFails(t *testing.T) {
	_, err := Decrypt(DeriveKey("dev-secret"), "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDecrypt))
}

func TestInvalidKeyLength(t *testing.T) {
	short := []byte("too-short-key")
	_, err := Encrypt(short, []byte("x"))
	require.Error(t, err)
	_, err = Decrypt(short, "c2lnaHRleHQ")
	require.Error(t, err)
}
