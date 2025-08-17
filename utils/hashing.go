package utils

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"github.com/go-crypt/crypt"
	"github.com/go-crypt/crypt/algorithm"
	"github.com/go-crypt/crypt/algorithm/argon2"
	"strings"
)

func CompareHash(password string, hashedPassword string) bool {
	var (
		valid bool
		err   error
	)

	if valid, err = crypt.CheckPassword(password, hashedPassword); err != nil {
		panic(err)
	}

	return valid
}

func HashPassword(password string) string {
	var (
		hasher *argon2.Hasher
		err    error
		digest algorithm.Digest
	)

	if hasher, err = argon2.New(
		argon2.WithProfileRFC9106LowMemory(),
	); err != nil {
		panic(err)
	}

	if digest, err = hasher.Hash(password); err != nil {
		panic(err)
	}

	return digest.Encode()
}

func CheapHash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}

func CheapCompareHash(input string, hash string) bool {
	inputHash := strings.TrimRight(CheapHash(input), "=")
	storedHash := strings.TrimRight(hash, "=")
	return subtle.ConstantTimeCompare([]byte(inputHash), []byte(storedHash)) == 1
}
