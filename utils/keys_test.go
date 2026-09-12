package utils

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodePublicKeyPem(t *testing.T, publicKey any) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func TestParseAndValidatePublicKeyPem_AcceptsEd25519(t *testing.T) {
	t.Parallel()
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	parsed, err := ParseAndValidatePublicKeyPem(encodePublicKeyPem(t, publicKey))

	require.NoError(t, err)
	assert.Equal(t, publicKey, parsed)
}

func TestParseAndValidatePublicKeyPem_AcceptsRsa2048(t *testing.T) {
	t.Parallel()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	_, err = ParseAndValidatePublicKeyPem(encodePublicKeyPem(t, &privateKey.PublicKey))

	require.NoError(t, err)
}

func TestParseAndValidatePublicKeyPem_RejectsRsa1024(t *testing.T) {
	t.Parallel()
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)

	_, err = ParseAndValidatePublicKeyPem(encodePublicKeyPem(t, &privateKey.PublicKey))

	require.ErrorIs(t, err, ErrHttpBadRequest)
	assert.ErrorContains(t, err, "1024 bits")
}

func TestParseAndValidatePublicKeyPem_RejectsEcdsa(t *testing.T) {
	t.Parallel()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	_, err = ParseAndValidatePublicKeyPem(encodePublicKeyPem(t, &privateKey.PublicKey))

	require.ErrorIs(t, err, ErrHttpBadRequest)
}

func TestParseAndValidatePublicKeyPem_RejectsGarbage(t *testing.T) {
	t.Parallel()

	_, err := ParseAndValidatePublicKeyPem("not a key")

	require.ErrorIs(t, err, ErrHttpBadRequest)
}

func TestValidatePublicKey_RejectsUnknownType(t *testing.T) {
	t.Parallel()

	err := ValidatePublicKey("not a key")

	require.ErrorIs(t, err, ErrHttpBadRequest)
}
