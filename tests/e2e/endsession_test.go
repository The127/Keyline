//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	endSessionAttackerVS  = "endsession-attacker-vs"
	endSessionSharedApp   = "endsession-shared-app"
	endSessionVictimURI   = "http://localhost:9001/victim-logged-out"
	endSessionAttackerURI = "http://localhost:9001/attacker-phish"
	endSessionUser        = "endsession-user"
	endSessionPassword    = "endsession-correct-horse-battery"
	// Within RFC 7636's 43-128 char window.
	endSessionPkceVerifier = "endsession-test-verifier-padding-padding-pad-1"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("OidcEndSession cross-tenant lookup ["+backend.name+"]", Ordered, func() {
			var h *harness

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				Expect(setupEndSessionFixtures(h)).To(Succeed())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			// Driving a fresh OIDC code flow per spec keeps each id_token
			// independent (the EXPLOIT spec destroys the user's session, but
			// the id_token_hint check itself doesn't need a live session, so
			// reuse across specs would still work — fresh tokens just keep
			// failure modes isolated).
			mintIdToken := func() string {
				challenge := authCodePkceChallenge(endSessionPkceVerifier)
				code, err := endSessionAuthCodeFlow(h.ApiUrl(), h.VirtualServer(), endSessionSharedApp, endSessionVictimURI, challenge)
				Expect(err).ToNot(HaveOccurred())

				form := url.Values{}
				form.Set("grant_type", "authorization_code")
				form.Set("code", code)
				form.Set("client_id", endSessionSharedApp)
				form.Set("redirect_uri", endSessionVictimURI)
				form.Set("code_verifier", endSessionPkceVerifier)

				resp, err := http.PostForm(fmt.Sprintf("%s/oidc/%s/token", h.ApiUrl(), h.VirtualServer()), form)
				Expect(err).ToNot(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				body := readJSON(resp)
				idTok, _ := body["id_token"].(string)
				Expect(idTok).ToNot(BeEmpty())
				return idTok
			}

			endSession := func(vs string, idTokenHint, postLogoutURI string) *http.Response {
				httpClient := &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
				q := url.Values{}
				q.Set("id_token_hint", idTokenHint)
				if postLogoutURI != "" {
					q.Set("post_logout_redirect_uri", postLogoutURI)
				}
				resp, err := httpClient.Get(fmt.Sprintf("%s/oidc/%s/end_session?%s", h.ApiUrl(), vs, q.Encode()))
				Expect(err).ToNot(HaveOccurred())
				return resp
			}

			Describe("GET /oidc/{vs}/end_session", func() {
				It("redirects to a post_logout_redirect_uri registered for THIS VS's app (happy path)", func() {
					idTok := mintIdToken()
					resp := endSession(h.VirtualServer(), idTok, endSessionVictimURI)
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(http.StatusFound))
					Expect(resp.Header.Get("Location")).To(HavePrefix(endSessionVictimURI))
				})

				It("rejects a post_logout_redirect_uri that is registered ONLY in another VS's same-named app (REGRESSION: cross-tenant lookup)", func() {
					// endSessionAttackerVS has an app with the same name as
					// the victim's, but its PostLogoutRedirectUris contains
					// endSessionAttackerURI which is NOT registered on
					// the victim VS's app. Pre-fix, the unscoped lookup
					// could return the attacker's row and pass the
					// whitelist check. Post-fix, the lookup is scoped to
					// the request VS, so the attacker's URI is never
					// considered.
					idTok := mintIdToken()
					resp := endSession(h.VirtualServer(), idTok, endSessionAttackerURI)
					defer resp.Body.Close()
					Expect(resp.StatusCode).ToNot(Equal(http.StatusFound),
						"end_session must not redirect to a URI only registered in another VS")
					Expect(resp.Header.Get("Location")).ToNot(ContainSubstring(endSessionAttackerURI))
				})

				It("rejects a post_logout_redirect_uri that is registered nowhere", func() {
					idTok := mintIdToken()
					resp := endSession(h.VirtualServer(), idTok, "http://localhost:9001/totally-unregistered")
					defer resp.Body.Close()
					Expect(resp.StatusCode).ToNot(Equal(http.StatusFound))
				})

				It("rejects an id_token_hint signed by a different VS than the request path", func() {
					// Mint an id_token against test-vs, then send it to
					// /oidc/<attacker-vs>/end_session. The keyfunc picks
					// the URL-path VS's keys, so the signature check must
					// fail.
					idTok := mintIdToken()
					resp := endSession(endSessionAttackerVS, idTok, endSessionAttackerURI)
					defer resp.Body.Close()
					Expect(resp.StatusCode).ToNot(Equal(http.StatusFound),
						"a token signed by another VS must not validate")
				})

				It("rejects when id_token_hint is missing", func() {
					resp := endSession(h.VirtualServer(), "", endSessionVictimURI)
					defer resp.Body.Close()
					Expect(resp.StatusCode).ToNot(Equal(http.StatusFound))
				})
			})
		})
	}
}

// endSessionAuthCodeFlow drives /authorize -> /verify-password -> /finish-login
// in a target VS, returning the authorization code. Mirrors authCodeFlow but
// parameterized over the VS name so it can be used against the attacker VS too.
func endSessionAuthCodeFlow(serverUrl, vs, clientId, redirectUri, codeChallenge string) (string, error) {
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	authorizeURL := fmt.Sprintf("%s/oidc/%s/authorize", serverUrl, vs)
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientId)
	q.Set("redirect_uri", redirectUri)
	q.Set("scope", "openid email profile")
	q.Set("state", "endsession-state")
	q.Set("nonce", "endsession-nonce")
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

	credentialsBody := fmt.Sprintf(`{"username":%q,"password":%q}`, endSessionUser, endSessionPassword)
	resp, err = httpClient.Post(
		fmt.Sprintf("%s/logins/%s/verify-password", serverUrl, loginToken),
		"application/json",
		strings.NewReader(credentialsBody),
	)
	if err != nil {
		return "", fmt.Errorf("verify-password: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("verify-password: status %d", resp.StatusCode)
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

func readJSON(resp *http.Response) map[string]any {
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		panic(fmt.Errorf("decode token response: %w", err))
	}
	return body
}

// setupEndSessionFixtures seeds two VSes with a same-named app (one
// per VS) plus a user with a password, such that the cross-tenant
// lookup is observable. The point of using two VSes with overlapping
// app names is precisely the precondition the bug needs.
func setupEndSessionFixtures(h *harness) error {
	scope := h.Scope().NewScope()
	defer scope.Close()

	ctx := middlewares.ContextWithScope(context.Background(), scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
	m := ioc.GetDependency[mediatr.Mediator](scope)
	dbContext := ioc.GetDependency[database.Context](scope)

	// Victim project + victim app on the harness's "test-vs".
	if _, err := mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: h.VirtualServer(),
		Slug:              "endsession-project",
		Name:              "End-Session Project",
	}); err != nil {
		return fmt.Errorf("victim project: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	if _, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      h.VirtualServer(),
		ProjectSlug:            "endsession-project",
		Name:                   endSessionSharedApp,
		DisplayName:            "End-Session Victim App",
		Type:                   repositories.ApplicationTypePublic,
		RedirectUris:           []string{endSessionVictimURI},
		PostLogoutRedirectUris: []string{endSessionVictimURI},
	}); err != nil {
		return fmt.Errorf("victim app: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	// Attacker VS with a same-named app and a different post_logout URI.
	if _, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:                    endSessionAttackerVS,
		DisplayName:             "End-Session Attacker VS",
		PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
	}); err != nil {
		return fmt.Errorf("attacker VS: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	if _, err := mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: endSessionAttackerVS,
		Slug:              "endsession-project",
		Name:              "End-Session Attacker Project",
	}); err != nil {
		return fmt.Errorf("attacker project: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	if _, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      endSessionAttackerVS,
		ProjectSlug:            "endsession-project",
		Name:                   endSessionSharedApp,
		DisplayName:            "End-Session Attacker App",
		Type:                   repositories.ApplicationTypePublic,
		RedirectUris:           []string{endSessionAttackerURI},
		PostLogoutRedirectUris: []string{endSessionAttackerURI},
	}); err != nil {
		return fmt.Errorf("attacker app: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	// Victim user.
	userResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: h.VirtualServer(),
		DisplayName:       "End-Session User",
		Username:          endSessionUser,
		Email:             endSessionUser + "@test.local",
		EmailVerified:     true,
	})
	if err != nil {
		return fmt.Errorf("victim user: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return err
	}

	cred := repositories.NewCredential(userResp.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: utils.HashPassword(endSessionPassword),
		Temporary:      false,
	})
	dbContext.Credentials().Insert(cred)
	return dbContext.SaveChanges(ctx)
}
