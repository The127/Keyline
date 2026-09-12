package utils

import (
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

const MinRsaPublicKeyBits = 2048

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

func ValidatePublicKey(publicKey any) error {
	switch key := publicKey.(type) {
	case *rsa.PublicKey:
		if key.N.BitLen() < MinRsaPublicKeyBits {
			return fmt.Errorf("rsa public key has %d bits, at least %d are required: %w", key.N.BitLen(), MinRsaPublicKeyBits, ErrHttpBadRequest)
		}
		return nil

	case ed25519.PublicKey:
		return nil

	default:
		return fmt.Errorf("unsupported public key type %T: %w", publicKey, ErrHttpBadRequest)
	}
}

func ParseAndValidatePublicKeyPem(publicKeyPem string) (any, error) {
	publicKey, err := ParsePublicKeyPem(publicKeyPem)
	if err != nil {
		return nil, err
	}

	err = ValidatePublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	return publicKey, nil
}
