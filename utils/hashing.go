package utils

import (
	"github.com/go-crypt/crypt"
	"github.com/go-crypt/crypt/algorithm"
	"github.com/go-crypt/crypt/algorithm/argon2"
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
