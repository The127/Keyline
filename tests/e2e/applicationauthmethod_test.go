//go:build e2e

package e2e

import (
	"context"

	"github.com/The127/Keyline/api"
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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	authMethodProjectSlug = "auth-method"
	authMethodRedirectUri = "http://localhost:9100/callback"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Application token endpoint auth method ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource)

				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: authMethodProjectSlug,
					Name: "Auth Method",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			createApplication := func(name string, type_ string, method *string) (api.CreateApplicationResponseDto, error) {
				return h.Client().Project().Application(authMethodProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:                    name,
					DisplayName:             name,
					Type:                    type_,
					RedirectUris:            []string{authMethodRedirectUri},
					TokenEndpointAuthMethod: method,
				})
			}

			getApplication := func(id api.CreateApplicationResponseDto) api.GetApplicationResponseDto {
				application, err := h.Client().Project().Application(authMethodProjectSlug).Get(h.Ctx(), id.Id)
				Expect(err).ToNot(HaveOccurred())
				return application
			}

			It("defaults a confidential application to client_secret and returns a secret", func() {
				created, err := createApplication("secret-app", "confidential", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(created.Secret).ToNot(BeNil())

				Expect(getApplication(created).TokenEndpointAuthMethod).To(HaveValue(Equal("client_secret")))
			})

			It("creates a private_key_jwt application without a secret", func() {
				created, err := createApplication("key-app", "confidential", utils.Ptr("private_key_jwt"))
				Expect(err).ToNot(HaveOccurred())
				Expect(created.Secret).To(BeNil())

				Expect(getApplication(created).TokenEndpointAuthMethod).To(HaveValue(Equal("private_key_jwt")))
			})

			It("leaves the method unset on a public application", func() {
				created, err := createApplication("public-app", "public", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(created.Secret).To(BeNil())

				Expect(getApplication(created).TokenEndpointAuthMethod).To(BeNil())
			})

			It("refuses a method on a public application", func() {
				_, err := createApplication("public-key-app", "public", utils.Ptr("private_key_jwt"))
				Expect(err).To(HaveOccurred())
			})

			It("refuses an unknown method", func() {
				_, err := createApplication("odd-app", "confidential", utils.Ptr("client_secret_jwt"))
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("Application token endpoint auth method from initial configuration ["+backend.name+"]", Ordered, func() {
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

			It("applies the method declared for an application in the initial virtual server", func() {
				scope := h.Scope().NewScope()
				defer utils.PanicOnError(scope.Close, "closing scope")
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)
				dbContext := ioc.GetDependency[database.Context](scope)

				const vsName = "auth-method-vs"
				created, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                    vsName,
					DisplayName:             "Auth Method VS",
					PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
					Projects: []commands.CreateVirtualServerProject{{
						Slug: authMethodProjectSlug,
						Name: "Auth Method",
						Applications: []commands.CreateVirtualServerProjectApplication{{
							Name:                    "key-app",
							DisplayName:             "Key App",
							Type:                    "confidential",
							RedirectUris:            []string{authMethodRedirectUri},
							TokenEndpointAuthMethod: utils.Ptr("private_key_jwt"),
						}},
					}},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(dbContext.SaveChanges(ctx)).To(Succeed())
				Expect(created).ToNot(BeNil())

				applications, _, err := dbContext.Applications().List(ctx, repositories.NewApplicationFilter().Name("key-app"))
				Expect(err).ToNot(HaveOccurred())
				Expect(applications).To(HaveLen(1))

				application, err := mediatr.Send[*queries.GetApplicationResult](ctx, m, queries.GetApplication{
					VirtualServerName: vsName,
					ProjectSlug:       authMethodProjectSlug,
					ApplicationId:     applications[0].Id(),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(application.TokenEndpointAuthMethod).To(HaveValue(Equal(repositories.TokenEndpointAuthMethodPrivateKeyJwt)))
				Expect(applications[0].HashedSecret()).To(BeEmpty())
			})
		})
	}
}
