package utils

import (
	"crypto/ed25519"
	"crypto/rand"
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

func GenerateKeyPair() (ed25519.PrivateKey, ed25519.PublicKey) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Errorf("failed to generate key pair: %w", err))
	}
	return privateKey, publicKey
}

func ExportPrivateKey(privateKey ed25519.PrivateKey) []byte {
	return privateKey
}

func ImportPrivateKey(privateKeyBytes []byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		panic(fmt.Errorf("invalid private key size: expected %d bytes, got %d bytes", ed25519.PrivateKeySize, len(privateKeyBytes)))
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return privateKey, publicKey
}
