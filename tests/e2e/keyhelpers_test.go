//go:build e2e

package e2e

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	. "github.com/onsi/gomega"
)

func rsaPublicKeyPem(bits int) string {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	Expect(err).ToNot(HaveOccurred())

	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	Expect(err).ToNot(HaveOccurred())

	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}
