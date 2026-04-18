package e2e

import (
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/config"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const wrongPrivateKey = "-----BEGIN PRIVATE KEY-----\nMFECAQEwBQYDK2VwBCIEIGlwiAvmqJTxz8n1Ewwwy7/XG2LJphqbOhfKcfg2l9YU\ngSEAi+MvQpVxlYdQrbbsn5xltPOYbU00qJtkEHPO2uzUmKQ=\n-----END PRIVATE KEY-----\n"

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Service user login ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("fails with wrong private key", func() {
				block, _ := pem.Decode([]byte(wrongPrivateKey))
				if block == nil {
					panic("failed to decode PEM")
				}

				key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
				Expect(err).ToNot(HaveOccurred())

				claims := jwt.MapClaims{
					"aud":    commands.AdminApplicationName,
					"iss":    serviceUserUsername,
					"sub":    serviceUserUsername,
					"scopes": "openid profile email",
				}
				jwtToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
				jwtToken.Header["kid"] = serviceUserKid
				signedJWT, err := jwtToken.SignedString(key)
				Expect(err).ToNot(HaveOccurred())

				resp, err := http.PostForm(fmt.Sprintf("%s/oidc/%s/token", h.ApiUrl(), h.VirtualServer()),
					url.Values{
						"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
						"subject_token":      {signedJWT},
						"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).ToNot(Equal(http.StatusOK))
			})

			It("exchanges token successfully", func() {
				block, _ := pem.Decode([]byte(serviceUserPrivateKey))
				if block == nil {
					panic("failed to decode PEM")
				}

				key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
				Expect(err).ToNot(HaveOccurred())

				claims := jwt.MapClaims{
					"aud":    commands.AdminApplicationName,
					"iss":    serviceUserUsername,
					"sub":    serviceUserUsername,
					"scopes": "openid profile email",
				}
				jwtToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
				jwtToken.Header["kid"] = serviceUserKid
				signedJWT, err := jwtToken.SignedString(key)
				Expect(err).ToNot(HaveOccurred())

				resp, err := http.PostForm(fmt.Sprintf("%s/oidc/%s/token", h.ApiUrl(), h.VirtualServer()),
					url.Values{
						"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
						"subject_token":      {signedJWT},
						"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var responseJson map[string]any
				err = json.NewDecoder(resp.Body).Decode(
					&responseJson,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(responseJson["access_token"]).ToNot(BeEmpty())
			})
		})
	}
}
