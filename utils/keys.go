package utils

import (
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func ParsePublicKeyPem(publicKeyPem string) (any, error) {
	block, _ := pem.Decode([]byte(publicKeyPem))
	if block == nil {
		return nil, fmt.Errorf("public key is not pem encoded: %w", ErrHttpBadRequest)
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("public key is not a pkix public key: %w", ErrHttpBadRequest)
	}

	switch publicKey.(type) {
	case *rsa.PublicKey, ed25519.PublicKey:
		return publicKey, nil
	default:
		return nil, fmt.Errorf("unsupported public key type %T: %w", publicKey, ErrHttpBadRequest)
	}
}
