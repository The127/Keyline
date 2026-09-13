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
	exchangeProjectSlug = "token-exchange"
	exchangeRedirectUri = "http://localhost:9300/callback"
	exchangerAppName    = "mungcp"
	targetAppName       = "cluster-a"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Trusted exchangers ["+backend.name+"]", Ordered, func() {
			var h *harness

			createApplication := func(name string, trustedExchangers []string) api.CreateApplicationResponseDto {
				created, err := h.Client().Project().Application(exchangeProjectSlug).Create(h.Ctx(), api.CreateApplicationRequestDto{
					Name:              name,
					DisplayName:       name,
					Type:              "public",
					RedirectUris:      []string{exchangeRedirectUri},
					TrustedExchangers: trustedExchangers,
				})
				Expect(err).ToNot(HaveOccurred())
				return created
			}

			getApplication := func(id api.CreateApplicationResponseDto) api.GetApplicationResponseDto {
				application, err := h.Client().Project().Application(exchangeProjectSlug).Get(h.Ctx(), id.Id)
				Expect(err).ToNot(HaveOccurred())
				return application
			}

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, serviceUserLogin)

				_, err := h.Client().Project().Create(h.Ctx(), api.CreateProjectRequestDto{
					Slug: exchangeProjectSlug,
					Name: "Token Exchange",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("trusts nobody by default", func() {
				created := createApplication("lonely-app", nil)

				Expect(getApplication(created).TrustedExchangers).To(BeEmpty())
			})

			It("stores the trusted exchangers on creation and changes them through patch", func() {
				created := createApplication(targetAppName, []string{exchangerAppName})
				Expect(getApplication(created).TrustedExchangers).To(ConsistOf(exchangerAppName))

				Expect(h.Client().Project().Application(exchangeProjectSlug).Patch(h.Ctx(), created.Id, api.PatchApplicationRequestDto{
					TrustedExchangers: []string{},
				})).To(Succeed())
				Expect(getApplication(created).TrustedExchangers).To(BeEmpty())

				Expect(h.Client().Project().Application(exchangeProjectSlug).Patch(h.Ctx(), created.Id, api.PatchApplicationRequestDto{
					DisplayName: utils.Ptr("Cluster A"),
				})).To(Succeed())
				Expect(getApplication(created).TrustedExchangers).To(BeEmpty())
			})
		})

		Describe("Trusted exchangers from initial configuration ["+backend.name+"]", Ordered, func() {
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

			It("applies the trusted exchangers declared for an application in the initial virtual server", func() {
				scope := h.Scope().NewScope()
				defer utils.PanicOnError(scope.Close, "closing scope")
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)
				dbContext := ioc.GetDependency[database.Context](scope)

				const vsName = "token-exchange-vs"
				_, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                    vsName,
					DisplayName:             "Token Exchange VS",
					PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
					Projects: []commands.CreateVirtualServerProject{{
						Slug: exchangeProjectSlug,
						Name: "Token Exchange",
						Applications: []commands.CreateVirtualServerProjectApplication{{
							Name:              targetAppName,
							DisplayName:       "Cluster A",
							Type:              "public",
							RedirectUris:      []string{exchangeRedirectUri},
							TrustedExchangers: []string{exchangerAppName},
						}},
					}},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(dbContext.SaveChanges(ctx)).To(Succeed())

				applications, _, err := dbContext.Applications().List(ctx, repositories.NewApplicationFilter().Name(targetAppName))
				Expect(err).ToNot(HaveOccurred())
				Expect(applications).To(HaveLen(1))

				application, err := mediatr.Send[*queries.GetApplicationResult](ctx, m, queries.GetApplication{
					VirtualServerName: vsName,
					ProjectSlug:       exchangeProjectSlug,
					ApplicationId:     applications[0].Id(),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(application.TrustedExchangers).To(ConsistOf(exchangerAppName))
			})
		})
	}
}
