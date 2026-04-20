//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Multi-algorithm signing keys ["+backend.name+"]", Ordered, func() {
			var h *harness
			const multiAlgVS = "multi-alg-vs"
			const patchTestVS = "patch-alg-vs"

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)

				scope := h.Scope().NewScope()
				defer scope.Close()
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)
				dbCtx := ioc.GetDependency[database.Context](scope)

				_, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                        multiAlgVS,
					DisplayName:                 "Multi Algorithm VS",
					PrimarySigningAlgorithm:     config.SigningAlgorithmEdDSA,
					AdditionalSigningAlgorithms: []config.SigningAlgorithm{config.SigningAlgorithmRS256},
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                    patchTestVS,
					DisplayName:             "Patch Test VS",
					PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(dbCtx.SaveChanges(ctx)).To(Succeed())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			Describe("JWKS endpoint", func() {
				It("returns a key for each configured algorithm", func() {
					resp, err := http.Get(jwksURL(h, multiAlgVS))
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close() //nolint:errcheck
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					var jwks jwksResponse
					Expect(json.NewDecoder(resp.Body).Decode(&jwks)).To(Succeed())

					Expect(jwks.Keys).To(HaveLen(2))
					Expect(jwkAlgs(jwks)).To(ConsistOf("EdDSA", "RS256"))
				})

				It("returns the correct JWK structure for EdDSA keys", func() {
					resp, err := http.Get(jwksURL(h, multiAlgVS))
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close() //nolint:errcheck

					var jwks jwksResponse
					Expect(json.NewDecoder(resp.Body).Decode(&jwks)).To(Succeed())

					eddsaKey := findJwkByAlg(jwks, "EdDSA")
					Expect(eddsaKey).ToNot(BeNil())
					Expect(eddsaKey["kty"]).To(Equal("OKP"))
					Expect(eddsaKey["crv"]).To(Equal("Ed25519"))
					Expect(eddsaKey["use"]).To(Equal("sig"))
					Expect(eddsaKey["kid"]).ToNot(BeEmpty())
					Expect(eddsaKey["x"]).ToNot(BeEmpty())
				})

				It("returns the correct JWK structure for RS256 keys", func() {
					resp, err := http.Get(jwksURL(h, multiAlgVS))
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close() //nolint:errcheck

					var jwks jwksResponse
					Expect(json.NewDecoder(resp.Body).Decode(&jwks)).To(Succeed())

					rs256Key := findJwkByAlg(jwks, "RS256")
					Expect(rs256Key).ToNot(BeNil())
					Expect(rs256Key["kty"]).To(Equal("RSA"))
					Expect(rs256Key["use"]).To(Equal("sig"))
					Expect(rs256Key["kid"]).ToNot(BeEmpty())
					Expect(rs256Key["n"]).ToNot(BeEmpty())
					Expect(rs256Key["e"]).ToNot(BeEmpty())
				})

				It("returns a single key for a single-algorithm VS", func() {
					resp, err := http.Get(jwksURL(h, h.VirtualServer()))
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close() //nolint:errcheck
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					var jwks jwksResponse
					Expect(json.NewDecoder(resp.Body).Decode(&jwks)).To(Succeed())

					Expect(jwks.Keys).To(HaveLen(1))
					Expect(jwks.Keys[0]["alg"]).To(Equal("EdDSA"))
				})

				It("keys for different algorithms have distinct KIDs", func() {
					resp, err := http.Get(jwksURL(h, multiAlgVS))
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close() //nolint:errcheck

					var jwks jwksResponse
					Expect(json.NewDecoder(resp.Body).Decode(&jwks)).To(Succeed())
					Expect(jwks.Keys).To(HaveLen(2))

					kid0 := jwks.Keys[0]["kid"].(string)
					kid1 := jwks.Keys[1]["kid"].(string)
					Expect(kid0).ToNot(Equal(kid1))
				})
			})

			Describe("discovery document", func() {
				It("lists all configured algorithms in id_token_signing_alg_values_supported", func() {
					resp, err := http.Get(discoveryURL(h, multiAlgVS))
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close() //nolint:errcheck
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					var doc struct {
						IdTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
					}
					Expect(json.NewDecoder(resp.Body).Decode(&doc)).To(Succeed())
					Expect(doc.IdTokenSigningAlgValuesSupported).To(ConsistOf("EdDSA", "RS256"))
				})

				It("lists only the primary algorithm for a single-algorithm VS", func() {
					resp, err := http.Get(discoveryURL(h, h.VirtualServer()))
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close() //nolint:errcheck

					var doc struct {
						IdTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
					}
					Expect(json.NewDecoder(resp.Body).Decode(&doc)).To(Succeed())
					Expect(doc.IdTokenSigningAlgValuesSupported).To(ConsistOf("EdDSA"))
				})
			})

			Describe("PATCH to add an additional algorithm", func() {
				It("immediately reflects the new key in JWKS", func() {
					respBefore, err := http.Get(jwksURL(h, patchTestVS))
					Expect(err).ToNot(HaveOccurred())
					defer respBefore.Body.Close() //nolint:errcheck
					var jwksBefore jwksResponse
					Expect(json.NewDecoder(respBefore.Body).Decode(&jwksBefore)).To(Succeed())
					Expect(jwksBefore.Keys).To(HaveLen(1))

					scope := h.Scope().NewScope()
					defer scope.Close()
					ctx := middlewares.ContextWithScope(context.Background(), scope)
					ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
					m := ioc.GetDependency[mediatr.Mediator](scope)
					dbCtx := ioc.GetDependency[database.Context](scope)

					additional := []config.SigningAlgorithm{config.SigningAlgorithmRS256}
					_, err = mediatr.Send[*commands.PatchVirtualServerResponse](ctx, m, commands.PatchVirtualServer{
						VirtualServerName:           patchTestVS,
						AdditionalSigningAlgorithms: &additional,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(dbCtx.SaveChanges(ctx)).To(Succeed())

					respAfter, err := http.Get(jwksURL(h, patchTestVS))
					Expect(err).ToNot(HaveOccurred())
					defer respAfter.Body.Close() //nolint:errcheck
					var jwksAfter jwksResponse
					Expect(json.NewDecoder(respAfter.Body).Decode(&jwksAfter)).To(Succeed())

					Expect(jwksAfter.Keys).To(HaveLen(2))
					Expect(jwkAlgs(jwksAfter)).To(ConsistOf("EdDSA", "RS256"))
				})
			})

			Describe("PATCH to remove an additional algorithm", func() {
				It("immediately removes the key from JWKS", func() {
					// patchTestVS was given RS256 as an additional algorithm by the prior test,
					// so it should currently have 2 keys. Patching to an empty additional list
					// should drop back to 1.
					scope := h.Scope().NewScope()
					defer scope.Close()
					ctx := middlewares.ContextWithScope(context.Background(), scope)
					ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
					m := ioc.GetDependency[mediatr.Mediator](scope)
					dbCtx := ioc.GetDependency[database.Context](scope)

					empty := []config.SigningAlgorithm{}
					_, err := mediatr.Send[*commands.PatchVirtualServerResponse](ctx, m, commands.PatchVirtualServer{
						VirtualServerName:           patchTestVS,
						AdditionalSigningAlgorithms: &empty,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(dbCtx.SaveChanges(ctx)).To(Succeed())

					resp, err := http.Get(jwksURL(h, patchTestVS))
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close() //nolint:errcheck
					var jwks jwksResponse
					Expect(json.NewDecoder(resp.Body).Decode(&jwks)).To(Succeed())

					Expect(jwks.Keys).To(HaveLen(1))
					Expect(jwks.Keys[0]["alg"]).To(Equal("EdDSA"))
				})
			})

			Describe("PATCH to change the primary algorithm", func() {
				It("updates the primary algorithm and discovery doc reflects the change", func() {
					scope := h.Scope().NewScope()
					defer scope.Close()
					ctx := middlewares.ContextWithScope(context.Background(), scope)
					ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
					m := ioc.GetDependency[mediatr.Mediator](scope)
					dbCtx := ioc.GetDependency[database.Context](scope)

					newPrimary := config.SigningAlgorithmRS256
					additional := []config.SigningAlgorithm{config.SigningAlgorithmEdDSA}
					_, err := mediatr.Send[*commands.PatchVirtualServerResponse](ctx, m, commands.PatchVirtualServer{
						VirtualServerName:           multiAlgVS,
						PrimarySigningAlgorithm:     &newPrimary,
						AdditionalSigningAlgorithms: &additional,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(dbCtx.SaveChanges(ctx)).To(Succeed())

					resp, err := http.Get(discoveryURL(h, multiAlgVS))
					Expect(err).ToNot(HaveOccurred())
					defer resp.Body.Close() //nolint:errcheck

					var doc struct {
						IdTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
					}
					Expect(json.NewDecoder(resp.Body).Decode(&doc)).To(Succeed())
					// RS256 should now be first (primary), EdDSA still present as additional
					Expect(doc.IdTokenSigningAlgValuesSupported[0]).To(Equal("RS256"))
					Expect(doc.IdTokenSigningAlgValuesSupported).To(ConsistOf("RS256", "EdDSA"))
				})
			})
		})
	}
}

type jwksResponse struct {
	Keys []map[string]any `json:"keys"`
}

func jwksURL(h *harness, vsName string) string {
	return fmt.Sprintf("%s/oidc/%s/.well-known/jwks.json", h.ApiUrl(), vsName)
}

func discoveryURL(h *harness, vsName string) string {
	return fmt.Sprintf("%s/oidc/%s/.well-known/openid-configuration", h.ApiUrl(), vsName)
}

func jwkAlgs(jwks jwksResponse) []string {
	algs := make([]string, len(jwks.Keys))
	for i, k := range jwks.Keys {
		algs[i] = k["alg"].(string)
	}
	return algs
}

func findJwkByAlg(jwks jwksResponse, alg string) map[string]any {
	for _, k := range jwks.Keys {
		if k["alg"] == alg {
			return k
		}
	}
	return nil
}
