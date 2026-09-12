//go:build e2e

package e2e

import (
	"context"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/google/uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	appKeysProjectSlug = "app-keys"
	appKeysRedirectUri = "http://localhost:9200/callback"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Application keys ["+backend.name+"]", Ordered, func() {
			var h *harness
			var keyAppId uuid.UUID
			var secretAppId uuid.UUID

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource)

				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: appKeysProjectSlug,
					Name: "App Keys",
				})
				Expect(err).ToNot(HaveOccurred())

				applications := h.Client().Project().Application(appKeysProjectSlug)
				keyApp, err := applications.Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:                    "key-app",
					DisplayName:             "Key App",
					Type:                    "confidential",
					RedirectUris:            []string{appKeysRedirectUri},
					TokenEndpointAuthMethod: utils.Ptr("private_key_jwt"),
				})
				Expect(err).ToNot(HaveOccurred())
				keyAppId = keyApp.Id

				secretApp, err := applications.Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:         "secret-app",
					DisplayName:  "Secret App",
					Type:         "confidential",
					RedirectUris: []string{appKeysRedirectUri},
				})
				Expect(err).ToNot(HaveOccurred())
				secretAppId = secretApp.Id
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			applications := func() client.ApplicationClient {
				return h.Client().Project().Application(appKeysProjectSlug)
			}

			It("starts without keys", func() {
				keys, err := applications().ListKeys(h.Ctx(), keyAppId)
				Expect(err).ToNot(HaveOccurred())
				Expect(keys).To(BeEmpty())
			})

			It("adds a key with the given kid and lists it", func() {
				added, err := applications().AddKey(h.Ctx(), keyAppId, api.AddApplicationKeyRequestDto{
					PublicKey: serviceUserPublicKey,
					Kid:       utils.Ptr("key-1"),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(added.Kid).To(Equal("key-1"))

				keys, err := applications().ListKeys(h.Ctx(), keyAppId)
				Expect(err).ToNot(HaveOccurred())
				Expect(keys).To(HaveLen(1))
				Expect(keys[0].Kid).To(Equal("key-1"))
				Expect(keys[0].PublicKey).To(Equal(serviceUserPublicKey))
			})

			It("generates a kid when none is given", func() {
				added, err := applications().AddKey(h.Ctx(), keyAppId, api.AddApplicationKeyRequestDto{
					PublicKey: serviceUserPublicKey,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(added.Kid).ToNot(BeEmpty())
				Expect(added.Kid).ToNot(Equal("key-1"))

				keys, err := applications().ListKeys(h.Ctx(), keyAppId)
				Expect(err).ToNot(HaveOccurred())
				Expect(keys).To(HaveLen(2))
			})

			It("refuses a duplicate kid", func() {
				_, err := applications().AddKey(h.Ctx(), keyAppId, api.AddApplicationKeyRequestDto{
					PublicKey: serviceUserPublicKey,
					Kid:       utils.Ptr("key-1"),
				})
				Expect(err).To(HaveOccurred())
			})

			It("refuses a key that is not a pem encoded public key", func() {
				_, err := applications().AddKey(h.Ctx(), keyAppId, api.AddApplicationKeyRequestDto{
					PublicKey: "not a key",
				})
				Expect(err).To(HaveOccurred())
			})

			It("refuses an RSA key shorter than 2048 bits", func() {
				_, err := applications().AddKey(h.Ctx(), keyAppId, api.AddApplicationKeyRequestDto{
					PublicKey: rsaPublicKeyPem(1024),
				})
				Expect(err).To(HaveOccurred())
			})

			It("accepts a 2048 bit RSA key", func() {
				added, err := applications().AddKey(h.Ctx(), keyAppId, api.AddApplicationKeyRequestDto{
					PublicKey: rsaPublicKeyPem(2048),
					Kid:       utils.Ptr("rsa-key"),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(added.Kid).To(Equal("rsa-key"))

				Expect(applications().RemoveKey(h.Ctx(), keyAppId, "rsa-key")).To(Succeed())
			})

			It("refuses a key on a client_secret application", func() {
				_, err := applications().AddKey(h.Ctx(), secretAppId, api.AddApplicationKeyRequestDto{
					PublicKey: serviceUserPublicKey,
				})
				Expect(err).To(HaveOccurred())
			})

			It("removes a key by kid", func() {
				Expect(applications().RemoveKey(h.Ctx(), keyAppId, "key-1")).To(Succeed())

				keys, err := applications().ListKeys(h.Ctx(), keyAppId)
				Expect(err).ToNot(HaveOccurred())
				Expect(keys).To(HaveLen(1))
				Expect(keys[0].Kid).ToNot(Equal("key-1"))
			})

			It("refuses to remove an unknown kid", func() {
				Expect(applications().RemoveKey(h.Ctx(), keyAppId, "key-1")).ToNot(Succeed())
			})
		})

		Describe("Application keys from initial configuration ["+backend.name+"]", Ordered, func() {
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

			It("stores the keys declared for an application in the initial virtual server", func() {
				scope := h.Scope().NewScope()
				defer utils.PanicOnError(scope.Close, "closing scope")
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)
				dbContext := ioc.GetDependency[database.Context](scope)

				const vsName = "app-keys-vs"
				_, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                    vsName,
					DisplayName:             "App Keys VS",
					PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
					Projects: []commands.CreateVirtualServerProject{{
						Slug: appKeysProjectSlug,
						Name: "App Keys",
						Applications: []commands.CreateVirtualServerProjectApplication{{
							Name:                    "key-app",
							DisplayName:             "Key App",
							Type:                    "confidential",
							RedirectUris:            []string{appKeysRedirectUri},
							TokenEndpointAuthMethod: utils.Ptr("private_key_jwt"),
							PublicKeys: []commands.CreateVirtualServerApplicationKey{
								{Pem: serviceUserPublicKey, Kid: "initial-1"},
								{Pem: serviceUserPublicKey, Kid: "initial-2"},
							},
						}},
					}},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(dbContext.SaveChanges(ctx)).To(Succeed())

				applications, _, err := dbContext.Applications().List(ctx, repositories.NewApplicationFilter().Name("key-app"))
				Expect(err).ToNot(HaveOccurred())
				Expect(applications).To(HaveLen(1))

				keys, err := mediatr.Send[*queries.ListApplicationKeysResponse](ctx, m, queries.ListApplicationKeys{
					VirtualServerName: vsName,
					ProjectSlug:       appKeysProjectSlug,
					ApplicationId:     applications[0].Id(),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(keys.Items).To(HaveLen(2))
				Expect(keys.Items[0].Kid).To(Equal("initial-1"))
				Expect(keys.Items[1].Kid).To(Equal("initial-2"))
			})

			It("refuses keys on a client_secret application in the initial virtual server", func() {
				scope := h.Scope().NewScope()
				defer utils.PanicOnError(scope.Close, "closing scope")
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)

				_, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                    "app-keys-bad-vs",
					DisplayName:             "App Keys Bad VS",
					PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
					Projects: []commands.CreateVirtualServerProject{{
						Slug: appKeysProjectSlug,
						Name: "App Keys",
						Applications: []commands.CreateVirtualServerProjectApplication{{
							Name:         "secret-app",
							DisplayName:  "Secret App",
							Type:         "confidential",
							HashedSecret: utils.Ptr("hashed"),
							RedirectUris: []string{appKeysRedirectUri},
							PublicKeys:   []commands.CreateVirtualServerApplicationKey{{Pem: serviceUserPublicKey, Kid: "k"}},
						}},
					}},
				})
				Expect(err).To(MatchError(ContainSubstring("private_key_jwt")))
			})
		})
	}
}
