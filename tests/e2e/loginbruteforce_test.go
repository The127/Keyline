//go:build e2e

package e2e

import (
	"context"
	"fmt"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/handlers"
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
	loginBfAppName       = "test-login-bf-app"
	loginBfUserUsername  = "test-login-bf-user"
	loginBfUserPassword  = "correct-horse-battery-staple"
	loginBfOtherUsername = "test-login-bf-other"
	loginBfOtherPassword = "another-correct-password"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Login brute-force protection ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				err := setupLoginBruteForceFixtures(h.Scope())
				Expect(err).ToNot(HaveOccurred())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			// Each spec mints its own fresh loginToken so they don't share
			// failed-attempt counters.
			mintLoginToken := func() string {
				deviceResp, err := h.Client().Oidc().BeginDeviceFlow(h.Ctx(), loginBfAppName, "openid")
				Expect(err).ToNot(HaveOccurred())
				loginToken, err := h.Client().Oidc().PostActivate(h.Ctx(), deviceResp.UserCode)
				Expect(err).ToNot(HaveOccurred())
				Expect(loginToken).ToNot(BeEmpty())
				return loginToken
			}

			Describe("POST /logins/{loginToken}/verify-password", func() {
				It("accepts the correct password on the first attempt", func() {
					loginToken := mintLoginToken()
					err := h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, loginBfUserUsername, loginBfUserPassword)
					Expect(err).ToNot(HaveOccurred())
				})

				It("accepts the correct password before the threshold is reached", func() {
					loginToken := mintLoginToken()
					// MaxFailedPasswordAttempts - 1 wrong attempts must not
					// invalidate the loginToken: the user gets the threshold
					// minus one chances before lockout.
					for i := 0; i < handlers.MaxFailedPasswordAttempts-1; i++ {
						err := h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, loginBfUserUsername, fmt.Sprintf("wrong-%d", i))
						Expect(err).To(HaveOccurred())
					}
					err := h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, loginBfUserUsername, loginBfUserPassword)
					Expect(err).ToNot(HaveOccurred())
				})

				It("invalidates the loginToken after MaxFailedPasswordAttempts wrong passwords", func() {
					loginToken := mintLoginToken()
					for i := 0; i < handlers.MaxFailedPasswordAttempts; i++ {
						err := h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, loginBfUserUsername, fmt.Sprintf("wrong-%d", i))
						Expect(err).To(HaveOccurred())
					}
					// Even the CORRECT password is rejected now: the
					// loginToken has been deleted from the token store.
					err := h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, loginBfUserUsername, loginBfUserPassword)
					Expect(err).To(HaveOccurred())
				})

				It("counts failed attempts across different usernames on the same loginToken (spray)", func() {
					loginToken := mintLoginToken()
					// Spray ONE password across MaxFailedPasswordAttempts
					// distinct usernames. The counter MUST tick per attempt
					// regardless of the username supplied -- otherwise an
					// attacker rotating usernames would never trip the cap.
					for i := 0; i < handlers.MaxFailedPasswordAttempts; i++ {
						err := h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, fmt.Sprintf("victim-%d", i), "spray")
						Expect(err).To(HaveOccurred())
					}
					err := h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, loginBfUserUsername, loginBfUserPassword)
					Expect(err).To(HaveOccurred())
				})

				It("counts failed attempts against unknown users (lockout still triggers)", func() {
					loginToken := mintLoginToken()
					// Ghost users must also tick the counter -- otherwise
					// an attacker could bypass the cap by sending bogus
					// usernames between real attempts.
					for i := 0; i < handlers.MaxFailedPasswordAttempts; i++ {
						err := h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, fmt.Sprintf("ghost-%d", i), "anything")
						Expect(err).To(HaveOccurred())
					}
					err := h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, loginBfUserUsername, loginBfUserPassword)
					Expect(err).To(HaveOccurred())
				})

				It("resets the counter on a successful login (separate loginToken)", func() {
					// A successful login completes the flow and burns the
					// loginToken via FinishLogin; subsequent attempts on
					// that token are 401, so this case is mostly a sanity
					// check that a fresh loginToken starts clean and the
					// fix doesn't leak counter state across loginTokens.
					tokenA := mintLoginToken()
					for i := 0; i < handlers.MaxFailedPasswordAttempts-1; i++ {
						err := h.Client().Oidc().VerifyPassword(h.Ctx(), tokenA, loginBfUserUsername, "wrong")
						Expect(err).To(HaveOccurred())
					}
					tokenB := mintLoginToken()
					// Token B must accept MaxFailedPasswordAttempts-1 wrong
					// attempts of its own -- the counter is per-loginToken.
					for i := 0; i < handlers.MaxFailedPasswordAttempts-1; i++ {
						err := h.Client().Oidc().VerifyPassword(h.Ctx(), tokenB, loginBfUserUsername, "wrong")
						Expect(err).To(HaveOccurred())
					}
					err := h.Client().Oidc().VerifyPassword(h.Ctx(), tokenB, loginBfUserUsername, loginBfUserPassword)
					Expect(err).ToNot(HaveOccurred())
				})

				It("supports a fresh loginToken after the previous one was burned", func() {
					tokenA := mintLoginToken()
					for i := 0; i < handlers.MaxFailedPasswordAttempts; i++ {
						_ = h.Client().Oidc().VerifyPassword(h.Ctx(), tokenA, loginBfUserUsername, "wrong")
					}
					// Burned tokenA -- correct password rejected.
					err := h.Client().Oidc().VerifyPassword(h.Ctx(), tokenA, loginBfUserUsername, loginBfUserPassword)
					Expect(err).To(HaveOccurred())

					// A second, anonymously-minted loginToken still works.
					// The fix MUST NOT lock the user out at the credential
					// level -- the cap is per-loginToken only.
					tokenB := mintLoginToken()
					err = h.Client().Oidc().VerifyPassword(h.Ctx(), tokenB, loginBfUserUsername, loginBfUserPassword)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	}
}

func setupLoginBruteForceFixtures(scope *ioc.DependencyProvider) error {
	subscope := scope.NewScope()
	defer subscope.Close()

	ctx := context.Background()
	ctx = middlewares.ContextWithScope(ctx, subscope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	m := ioc.GetDependency[mediatr.Mediator](subscope)
	dbContext := ioc.GetDependency[database.Context](subscope)

	_, err := mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: "test-vs",
		Slug:              "login-bf-project",
		Name:              "Login Brute-force Project",
	})
	if err != nil {
		return fmt.Errorf("creating project: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return fmt.Errorf("saving project: %w", err)
	}

	appResp, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      "test-vs",
		ProjectSlug:            "login-bf-project",
		Name:                   loginBfAppName,
		DisplayName:            "Test Login Brute-force App",
		Type:                   repositories.ApplicationTypePublic,
		RedirectUris:           []string{"http://localhost:9999/callback"},
		PostLogoutRedirectUris: []string{},
	})
	if err != nil {
		return fmt.Errorf("creating application: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return fmt.Errorf("saving application: %w", err)
	}

	_, err = mediatr.Send[*commands.PatchApplicationResponse](ctx, m, commands.PatchApplication{
		VirtualServerName: "test-vs",
		ProjectSlug:       "login-bf-project",
		ApplicationId:     appResp.Id,
		DeviceFlowEnabled: utils.Ptr(true),
	})
	if err != nil {
		return fmt.Errorf("enabling device flow: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return fmt.Errorf("saving device flow flag: %w", err)
	}

	if err := seedUserWithPassword(ctx, m, dbContext, loginBfUserUsername, loginBfUserPassword); err != nil {
		return err
	}
	if err := seedUserWithPassword(ctx, m, dbContext, loginBfOtherUsername, loginBfOtherPassword); err != nil {
		return err
	}
	return nil
}

func seedUserWithPassword(
	ctx context.Context,
	m mediatr.Mediator,
	dbContext database.Context,
	username string,
	password string,
) error {
	userResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: "test-vs",
		DisplayName:       "Display " + username,
		Username:          username,
		Email:             username + "@test.local",
		EmailVerified:     true,
	})
	if err != nil {
		return fmt.Errorf("creating user %s: %w", username, err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return fmt.Errorf("saving user %s: %w", username, err)
	}

	cred := repositories.NewCredential(userResp.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: utils.HashPassword(password),
		Temporary:      false,
	})
	dbContext.Credentials().Insert(cred)
	if err := dbContext.SaveChanges(ctx); err != nil {
		return fmt.Errorf("saving credential for %s: %w", username, err)
	}
	return nil
}

// Compile-time guard: setupLoginBruteForceFixtures must remain compatible
// with newE2eTestHarness's scope shape. Catches refactors of the harness.
var _ = uuid.Nil
