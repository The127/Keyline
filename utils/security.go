package utils

import (
	"crypto/rand"
)

func GetSecureRandomBytes(length int) []byte {
	bytes := make([]byte, length)
	_, _ = rand.Read(bytes)
	return bytes
}
