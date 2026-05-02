//go:build e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

// Regression tests for the "authorization code is not bound to client/
// redirect_uri/PKCE" vulnerability. Each scenario drives the full login flow
// and then exercises one specific check at /token.

const (
	authCodePublicAppName       = "auth-code-public-app"
	authCodeConfidentialAppName = "auth-code-confidential-app"
	authCodeConfidentialSecret  = "test-confidential-secret-for-e2e-do-not-reuse"
	authCodeRedirect            = "http://localhost:9000/callback"
	authCodeOtherRedirect       = "http://localhost:9000/other-callback"
	authCodeUserName            = "auth-code-user"
	authCodeUserPassword        = "auth-code-user-password-1"

	// 56 chars; in RFC 7636's 43-128 window.
	authCodePkceVerifier = "regression-test-verifier-do-not-use-in-real-systems-x12"
)

func authCodePkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// authCodeFlow drives /authorize -> /logins/<token>/verify-password -> /finish-login
// and returns the issued authorization code.
func authCodeFlow(serverUrl, clientId, redirectUri, codeChallenge string) (string, error) {
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	authorizeURL := fmt.Sprintf("%s/oidc/test-vs/authorize", serverUrl)
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientId)
	q.Set("redirect_uri", redirectUri)
	q.Set("scope", "openid email profile")
	q.Set("state", "regression-state")
	q.Set("nonce", "regression-nonce")
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")

	resp, err := httpClient.Get(authorizeURL + "?" + q.Encode())
	if err != nil {
		return "", fmt.Errorf("authorize: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("authorize: expected 302, got %d", resp.StatusCode)
	}

	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return "", fmt.Errorf("parsing login redirect: %w", err)
	}
	loginToken := loc.Query().Get("token")
	if loginToken == "" {
		return "", fmt.Errorf("no login token in /authorize redirect: %s", resp.Header.Get("Location"))
	}

	credentialsBody := fmt.Sprintf(`{"username":%q,"password":%q}`, authCodeUserName, authCodeUserPassword)
	resp, err = httpClient.Post(
		fmt.Sprintf("%s/logins/%s/verify-password", serverUrl, loginToken),
		"application/json",
		strings.NewReader(credentialsBody),
	)
	if err != nil {
		return "", fmt.Errorf("verify-password: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("verify-password: status %d: %s", resp.StatusCode, body)
	}

	resp, err = httpClient.Post(fmt.Sprintf("%s/logins/%s/finish-login", serverUrl, loginToken), "", nil)
	if err != nil {
		return "", fmt.Errorf("finish-login: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("finish-login: expected 302, got %d", resp.StatusCode)
	}
	cookieHdr := resp.Header.Get("Set-Cookie")
	if cookieHdr == "" {
		return "", fmt.Errorf("finish-login: no Set-Cookie")
	}
	sessionCookie := strings.SplitN(cookieHdr, ";", 2)[0]
	nextURL := resp.Header.Get("Location")
	if !strings.HasPrefix(nextURL, "http") {
		nextURL = serverUrl + nextURL
	}

	req, err := http.NewRequest(http.MethodGet, nextURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", sessionCookie)
	resp, err = httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("second authorize: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("second authorize: expected 302, got %d", resp.StatusCode)
	}

	finalURL, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return "", fmt.Errorf("parsing final redirect: %w", err)
	}
	if errStr := finalURL.Query().Get("error"); errStr != "" {
		return "", fmt.Errorf("authorize returned error: %s (%s)", errStr, finalURL.Query().Get("error_description"))
	}
	code := finalURL.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("no code in final redirect: %s", resp.Header.Get("Location"))
	}
	return code, nil
}

// postToken posts form-encoded params to the token endpoint and returns the parsed JSON.
func postToken(serverUrl string, form url.Values) (int, map[string]any, error) {
	resp, err := http.PostForm(fmt.Sprintf("%s/oidc/test-vs/token", serverUrl), form)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	parsed := map[string]any{}
	_ = json.Unmarshal(body, &parsed)
	return resp.StatusCode, parsed, nil
}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Authorization Code Flow regression ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				Expect(setupAuthCodeFixtures(h.Scope())).To(Succeed())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			Describe("/authorize", func() {
				It("rejects requests without a code_challenge", func() {
					httpClient := &http.Client{
						CheckRedirect: func(req *http.Request, via []*http.Request) error {
							return http.ErrUseLastResponse
						},
					}
					q := url.Values{}
					q.Set("response_type", "code")
					q.Set("client_id", authCodePublicAppName)
					q.Set("redirect_uri", authCodeRedirect)
					q.Set("scope", "openid")
					resp, err := httpClient.Get(fmt.Sprintf("%s/oidc/test-vs/authorize?%s", h.ApiUrl(), q.Encode()))
					Expect(err).ToNot(HaveOccurred())
					resp.Body.Close()
					// Errors are reported by redirecting to the redirect_uri with ?error=...
					Expect(resp.StatusCode).To(Equal(http.StatusFound))
					loc, err := url.Parse(resp.Header.Get("Location"))
					Expect(err).ToNot(HaveOccurred())
					Expect(loc.Query().Get("error")).To(Equal("invalid_request"))
				})

				It("rejects code_challenge_method=plain", func() {
					httpClient := &http.Client{
						CheckRedirect: func(req *http.Request, via []*http.Request) error {
							return http.ErrUseLastResponse
						},
					}
					q := url.Values{}
					q.Set("response_type", "code")
					q.Set("client_id", authCodePublicAppName)
					q.Set("redirect_uri", authCodeRedirect)
					q.Set("scope", "openid")
					q.Set("code_challenge", authCodePkceVerifier)
					q.Set("code_challenge_method", "plain")
					resp, err := httpClient.Get(fmt.Sprintf("%s/oidc/test-vs/authorize?%s", h.ApiUrl(), q.Encode()))
					Expect(err).ToNot(HaveOccurred())
					resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(http.StatusFound))
					loc, err := url.Parse(resp.Header.Get("Location"))
					Expect(err).ToNot(HaveOccurred())
					Expect(loc.Query().Get("error")).To(Equal("invalid_request"))
				})
			})

			Describe("/token authorization_code", func() {
				It("legitimate redemption with PKCE verifier succeeds", func() {
					code, err := authCodeFlow(h.ApiUrl(), authCodePublicAppName, authCodeRedirect, authCodePkceChallenge(authCodePkceVerifier))
					Expect(err).ToNot(HaveOccurred())

					form := url.Values{}
					form.Set("grant_type", "authorization_code")
					form.Set("code", code)
					form.Set("client_id", authCodePublicAppName)
					form.Set("redirect_uri", authCodeRedirect)
					form.Set("code_verifier", authCodePkceVerifier)

					status, body, err := postToken(h.ApiUrl(), form)
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(http.StatusOK), fmt.Sprintf("body: %v", body))
					Expect(body["access_token"]).ToNot(BeNil())
				})

				It("rejects when code_verifier is missing (REGRESSION: PKCE bypass)", func() {
					code, err := authCodeFlow(h.ApiUrl(), authCodePublicAppName, authCodeRedirect, authCodePkceChallenge(authCodePkceVerifier))
					Expect(err).ToNot(HaveOccurred())

					form := url.Values{}
					form.Set("grant_type", "authorization_code")
					form.Set("code", code)
					form.Set("client_id", authCodePublicAppName)
					form.Set("redirect_uri", authCodeRedirect)
					// NO code_verifier
					status, body, err := postToken(h.ApiUrl(), form)
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_grant"))
				})

				It("rejects a wrong code_verifier (REGRESSION: PKCE not actually checked)", func() {
					code, err := authCodeFlow(h.ApiUrl(), authCodePublicAppName, authCodeRedirect, authCodePkceChallenge(authCodePkceVerifier))
					Expect(err).ToNot(HaveOccurred())

					form := url.Values{}
					form.Set("grant_type", "authorization_code")
					form.Set("code", code)
					form.Set("client_id", authCodePublicAppName)
					form.Set("redirect_uri", authCodeRedirect)
					form.Set("code_verifier", "this-verifier-is-wrong-but-still-the-right-length-okay")

					status, body, err := postToken(h.ApiUrl(), form)
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_grant"))
				})

				It("rejects redemption with a different client_id (REGRESSION: code/client binding)", func() {
					code, err := authCodeFlow(h.ApiUrl(), authCodePublicAppName, authCodeRedirect, authCodePkceChallenge(authCodePkceVerifier))
					Expect(err).ToNot(HaveOccurred())

					form := url.Values{}
					form.Set("grant_type", "authorization_code")
					form.Set("code", code)
					// Code was issued to public app; redeem as the confidential app, fully authenticated.
					form.Set("client_id", authCodeConfidentialAppName)
					form.Set("client_secret", authCodeConfidentialSecret)
					form.Set("redirect_uri", authCodeRedirect)
					form.Set("code_verifier", authCodePkceVerifier)

					status, body, err := postToken(h.ApiUrl(), form)
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_grant"))
				})

				It("rejects a mismatched redirect_uri (REGRESSION: code/redirect binding)", func() {
					code, err := authCodeFlow(h.ApiUrl(), authCodePublicAppName, authCodeRedirect, authCodePkceChallenge(authCodePkceVerifier))
					Expect(err).ToNot(HaveOccurred())

					form := url.Values{}
					form.Set("grant_type", "authorization_code")
					form.Set("code", code)
					form.Set("client_id", authCodePublicAppName)
					form.Set("redirect_uri", authCodeOtherRedirect) // different from /authorize
					form.Set("code_verifier", authCodePkceVerifier)

					status, body, err := postToken(h.ApiUrl(), form)
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_grant"))
				})

				It("rejects confidential client with empty client_secret (REGRESSION: empty-secret bypass)", func() {
					// Code is issued to the confidential client; we then try to redeem
					// it without the secret. Pre-fix this returned tokens.
					code, err := authCodeFlow(h.ApiUrl(), authCodeConfidentialAppName, authCodeRedirect, authCodePkceChallenge(authCodePkceVerifier))
					Expect(err).ToNot(HaveOccurred())

					form := url.Values{}
					form.Set("grant_type", "authorization_code")
					form.Set("code", code)
					form.Set("client_id", authCodeConfidentialAppName)
					form.Set("redirect_uri", authCodeRedirect)
					form.Set("code_verifier", authCodePkceVerifier)
					// NO client_secret

					status, body, err := postToken(h.ApiUrl(), form)
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})

				It("rejects confidential client with wrong client_secret", func() {
					code, err := authCodeFlow(h.ApiUrl(), authCodeConfidentialAppName, authCodeRedirect, authCodePkceChallenge(authCodePkceVerifier))
					Expect(err).ToNot(HaveOccurred())

					form := url.Values{}
					form.Set("grant_type", "authorization_code")
					form.Set("code", code)
					form.Set("client_id", authCodeConfidentialAppName)
					form.Set("client_secret", "the-wrong-secret-here-it-is-not-correct")
					form.Set("redirect_uri", authCodeRedirect)
					form.Set("code_verifier", authCodePkceVerifier)

					status, body, err := postToken(h.ApiUrl(), form)
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})

				It("rejects public client redemption that includes a client_secret", func() {
					code, err := authCodeFlow(h.ApiUrl(), authCodePublicAppName, authCodeRedirect, authCodePkceChallenge(authCodePkceVerifier))
					Expect(err).ToNot(HaveOccurred())

					form := url.Values{}
					form.Set("grant_type", "authorization_code")
					form.Set("code", code)
					form.Set("client_id", authCodePublicAppName)
					form.Set("client_secret", "anything-since-public-clients-have-no-secret")
					form.Set("redirect_uri", authCodeRedirect)
					form.Set("code_verifier", authCodePkceVerifier)

					status, body, err := postToken(h.ApiUrl(), form)
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})
			})
		})
	}
}

func setupAuthCodeFixtures(scope *ioc.DependencyProvider) error {
	subscope := scope.NewScope()
	defer subscope.Close()

	ctx := context.Background()
	ctx = middlewares.ContextWithScope(ctx, subscope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	m := ioc.GetDependency[mediatr.Mediator](subscope)
	dbContext := ioc.GetDependency[database.Context](subscope)

	if _, err := mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: "test-vs",
		Slug:              "auth-code-project",
		Name:              "Auth Code Project",
	}); err != nil {
		return fmt.Errorf("creating project: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	if _, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      "test-vs",
		ProjectSlug:            "auth-code-project",
		Name:                   authCodePublicAppName,
		DisplayName:            "Auth Code Public App",
		Type:                   repositories.ApplicationTypePublic,
		RedirectUris:           []string{authCodeRedirect, authCodeOtherRedirect},
		PostLogoutRedirectUris: []string{},
	}); err != nil {
		return fmt.Errorf("creating public app: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	if _, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      "test-vs",
		ProjectSlug:            "auth-code-project",
		Name:                   authCodeConfidentialAppName,
		DisplayName:            "Auth Code Confidential App",
		Type:                   repositories.ApplicationTypeConfidential,
		HashedSecret:           utils.Ptr(utils.CheapHash(authCodeConfidentialSecret)),
		RedirectUris:           []string{authCodeRedirect, authCodeOtherRedirect},
		PostLogoutRedirectUris: []string{},
	}); err != nil {
		return fmt.Errorf("creating confidential app: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	userResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: "test-vs",
		DisplayName:       "Auth Code User",
		Username:          authCodeUserName,
		Email:             authCodeUserName + "@test.local",
		EmailVerified:     true,
	})
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	cred := repositories.NewCredential(userResp.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: utils.HashPassword(authCodeUserPassword),
		Temporary:      false,
	})
	dbContext.Credentials().Insert(cred)
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	_ = uuid.Nil // keep uuid import
	return nil
}
