package utils

import (
	"Keyline/internal/config"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"fmt"
)

func GenerateCodeFromBytes(bytes []byte) string {
	return fmt.Sprintf("%x", bytes)[:6]
}

func GetSecureRandomBytes(length int) []byte {
	bytes := make([]byte, length)
	_, _ = rand.Read(bytes)
	return bytes
}

func ImportPrivateKey(privateKeyBytes []byte, algorithm config.SigningAlgorithm) (any, any) {
	switch algorithm {
	case config.SigningAlgorithmEdDSA:
		if len(privateKeyBytes) != ed25519.PrivateKeySize {
			panic(fmt.Errorf("invalid private key size: expected %d bytes, got %d bytes", ed25519.PrivateKeySize, len(privateKeyBytes)))
		}

		privateKey := ed25519.PrivateKey(privateKeyBytes)
		publicKey := privateKey.Public().(ed25519.PublicKey)
		return privateKey, publicKey

	case config.SigningAlgorithmRS256:
		privKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
		if err != nil {
			panic(err)
		}
		return privKey, privKey.Public()

	default:
		panic(fmt.Errorf("invalid signing algorithm: %s", algorithm))
	}
}
