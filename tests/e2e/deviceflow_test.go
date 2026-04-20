//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	deviceAppName      = "test-device-app"
	deviceUserUsername = "test-device-user"
	deviceUserPassword = "test-device-password-1"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Device Authorization Grant ["+backend.name+"]", Ordered, func() {
			var h *harness
			var deviceAppId uuid.UUID

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				var err error
				deviceAppId, err = setupDeviceFlowFixtures(h.Scope())
				Expect(err).ToNot(HaveOccurred())
				_ = deviceAppId
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			Describe("POST /device", func() {
				It("rejects missing client_id", func() {
					_, err := h.Client().Oidc().BeginDeviceFlow(h.Ctx(), "", "openid")
					Expect(err).To(HaveOccurred())
				})

				It("rejects application without device flow enabled", func() {
					_, err := h.Client().Oidc().BeginDeviceFlow(h.Ctx(), commands.AdminApplicationName, "openid")
					Expect(err).To(HaveOccurred())
				})

				It("rejects missing openid scope", func() {
					_, err := h.Client().Oidc().BeginDeviceFlow(h.Ctx(), deviceAppName, "profile")
					Expect(err).To(HaveOccurred())
				})

				It("returns device authorization response", func() {
					resp, err := h.Client().Oidc().BeginDeviceFlow(h.Ctx(), deviceAppName, "openid")
					Expect(err).ToNot(HaveOccurred())
					Expect(resp.DeviceCode).ToNot(BeEmpty())
					Expect(resp.UserCode).ToNot(BeEmpty())
					Expect(resp.VerificationUri).ToNot(BeEmpty())
					Expect(resp.VerificationUriComplete).ToNot(BeEmpty())
					Expect(resp.ExpiresIn).To(BeEquivalentTo(600))
					Expect(resp.Interval).To(BeEquivalentTo(5))
				})
			})

			Describe("POST /activate", func() {
				It("rejects unknown user_code", func() {
					_, err := h.Client().Oidc().PostActivate(h.Ctx(), "XXXX-XXXX")
					Expect(err).To(MatchError(client.ErrInvalidUserCode))
				})
			})

			Describe("Full device authorization flow", func() {
				It("issues tokens after user approves", func() {
					// Step 1: CLI requests device authorization
					deviceResp, err := h.Client().Oidc().BeginDeviceFlow(h.Ctx(), deviceAppName, "openid")
					Expect(err).ToNot(HaveOccurred())
					Expect(deviceResp.DeviceCode).ToNot(BeEmpty())
					Expect(deviceResp.UserCode).ToNot(BeEmpty())

					// Step 2: Poll while pending
					_, pollErr := h.Client().Oidc().PollDeviceToken(h.Ctx(), deviceAppName, deviceResp.DeviceCode)
					Expect(pollErr).To(MatchError(client.ErrAuthorizationPending))

					// Step 3: User submits user_code on activation page
					loginToken, err := h.Client().Oidc().PostActivate(h.Ctx(), deviceResp.UserCode)
					Expect(err).ToNot(HaveOccurred())
					Expect(loginToken).ToNot(BeEmpty())

					// Step 4: User completes login (verify password)
					err = h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, deviceUserUsername, deviceUserPassword)
					Expect(err).ToNot(HaveOccurred())

					// Step 5: Finish login — marks device code as authorized
					err = h.Client().Oidc().FinishLogin(h.Ctx(), loginToken)
					Expect(err).ToNot(HaveOccurred())

					// Step 6: CLI polls and receives tokens
					tokenResp, err := h.Client().Oidc().PollDeviceToken(h.Ctx(), deviceAppName, deviceResp.DeviceCode)
					Expect(err).ToNot(HaveOccurred())
					Expect(tokenResp.AccessToken).ToNot(BeEmpty())
					Expect(tokenResp.IdToken).ToNot(BeEmpty())
					Expect(tokenResp.RefreshToken).ToNot(BeEmpty())
					Expect(tokenResp.TokenType).To(Equal("Bearer"))
				})

				It("rejects double use of device_code", func() {
					deviceResp, err := h.Client().Oidc().BeginDeviceFlow(h.Ctx(), deviceAppName, "openid")
					Expect(err).ToNot(HaveOccurred())

					loginToken, err := h.Client().Oidc().PostActivate(h.Ctx(), deviceResp.UserCode)
					Expect(err).ToNot(HaveOccurred())

					err = h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, deviceUserUsername, deviceUserPassword)
					Expect(err).ToNot(HaveOccurred())

					err = h.Client().Oidc().FinishLogin(h.Ctx(), loginToken)
					Expect(err).ToNot(HaveOccurred())

					// First poll succeeds
					_, err = h.Client().Oidc().PollDeviceToken(h.Ctx(), deviceAppName, deviceResp.DeviceCode)
					Expect(err).ToNot(HaveOccurred())

					// Second poll fails — one-time use
					_, err = h.Client().Oidc().PollDeviceToken(h.Ctx(), deviceAppName, deviceResp.DeviceCode)
					Expect(err).To(MatchError(client.ErrExpiredToken))
				})

				It("returns expired_token for unknown device_code", func() {
					_, err := h.Client().Oidc().PollDeviceToken(h.Ctx(), deviceAppName, "nonexistent-device-code")
					Expect(err).To(MatchError(client.ErrExpiredToken))
				})
			})
		})
	}
}

func setupDeviceFlowFixtures(scope *ioc.DependencyProvider) (uuid.UUID, error) {
	subscope := scope.NewScope()
	defer subscope.Close()

	ctx := context.Background()
	ctx = middlewares.ContextWithScope(ctx, subscope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	m := ioc.GetDependency[mediatr.Mediator](subscope)
	dbContext := ioc.GetDependency[database.Context](subscope)

	_, err := mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: "test-vs",
		Slug:              "device-flow-project",
		Name:              "Device Flow Project",
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("creating project: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("saving project: %w", err)
	}

	appResp, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      "test-vs",
		ProjectSlug:            "device-flow-project",
		Name:                   deviceAppName,
		DisplayName:            "Test Device App",
		Type:                   repositories.ApplicationTypePublic,
		RedirectUris:           []string{"http://localhost:9999/callback"},
		PostLogoutRedirectUris: []string{},
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("creating application: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("saving application: %w", err)
	}

	_, err = mediatr.Send[*commands.PatchApplicationResponse](ctx, m, commands.PatchApplication{
		VirtualServerName: "test-vs",
		ProjectSlug:       "device-flow-project",
		ApplicationId:     appResp.Id,
		DeviceFlowEnabled: utils.Ptr(true),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("enabling device flow: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("saving device flow flag: %w", err)
	}

	userResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: "test-vs",
		DisplayName:       "Test Device User",
		Username:          deviceUserUsername,
		Email:             deviceUserUsername + "@test.local",
		EmailVerified:     true,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("creating user: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("saving user: %w", err)
	}

	passwordCred := repositories.NewCredential(userResp.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: utils.HashPassword(deviceUserPassword),
		Temporary:      false,
	})
	dbContext.Credentials().Insert(passwordCred)
	if err := dbContext.SaveChanges(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("saving password credential: %w", err)
	}

	return appResp.Id, nil
}
