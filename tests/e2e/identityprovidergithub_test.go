//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"

	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/client"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	stubGithubClientId     = "gh-client"
	stubGithubClientSecret = "gh-secret"
)

type stubGithubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

type stubGithubUser struct {
	Id     int               `json:"id"`
	Login  string            `json:"login"`
	Name   string            `json:"name"`
	Email  *string           `json:"email"`
	Emails []stubGithubEmail `json:"-"`
}

type stubGithub struct {
	server *httptest.Server
	mu     sync.Mutex
	user   stubGithubUser
	codes  map[string]stubGithubUser
	tokens map[string]stubGithubUser
}

func newStubGithub() *stubGithub {
	stub := &stubGithub{
		codes:  map[string]stubGithubUser{},
		tokens: map[string]stubGithubUser{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/authorize", stub.authorize)
	mux.HandleFunc("/login/oauth/access_token", stub.accessToken)
	mux.HandleFunc("/user", stub.userinfo)
	mux.HandleFunc("/user/emails", stub.userEmails)
	stub.server = httptest.NewServer(mux)
	return stub
}

func (s *stubGithub) actAs(user stubGithubUser) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.user = user
}

func (s *stubGithub) authorize(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := r.URL.Query()
	if query.Get("client_id") != stubGithubClientId || query.Get("response_type") != "code" {
		http.Error(w, "bad authorize request", http.StatusBadRequest)
		return
	}

	code := fmt.Sprintf("code-%d", len(s.codes)+1)
	s.codes[code] = s.user
	redirect, _ := url.Parse(query.Get("redirect_uri"))
	back := redirect.Query()
	back.Set("code", code)
	back.Set("state", query.Get("state"))
	redirect.RawQuery = back.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (s *stubGithub) accessToken(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = r.ParseForm()
	user, ok := s.codes[r.Form.Get("code")]
	if !ok || r.Form.Get("client_id") != stubGithubClientId || r.Form.Get("client_secret") != stubGithubClientSecret {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "bad_verification_code"})
		return
	}

	delete(s.codes, r.Form.Get("code"))
	token := fmt.Sprintf("token-%d", len(s.tokens)+1)
	s.tokens[token] = user
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": token, "token_type": "bearer", "scope": "read:user,user:email"})
}

func (s *stubGithub) bearer(r *http.Request) (stubGithubUser, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	return user, ok
}

func (s *stubGithub) userinfo(w http.ResponseWriter, r *http.Request) {
	user, ok := s.bearer(r)
	if !ok {
		http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (s *stubGithub) userEmails(w http.ResponseWriter, r *http.Request) {
	user, ok := s.bearer(r)
	if !ok {
		http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user.Emails)
}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Identity provider login at github ["+backend.name+"]", Ordered, func() {
			var h *harness
			var github *stubGithub

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}

				h = newE2eTestHarness(backend.dbMode, serviceUserTokenSource)
				github = newStubGithub()

				_, err := h.Client().VirtualServer().IdentityProviders().Create(h.Ctx(), api.CreateIdentityProviderRequestDto{
					Name:                  "github",
					DisplayName:           "GitHub",
					Preset:                "github",
					AuthorizationEndpoint: github.server.URL + "/login/oauth/authorize",
					TokenEndpoint:         github.server.URL + "/login/oauth/access_token",
					UserinfoEndpoint:      github.server.URL + "/user",
					ClientId:              stubGithubClientId,
					ClientSecret:          stubGithubClientSecret,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(h.Client().VirtualServer().Patch(h.Ctx(), client.PatchVirtualServerInput{EnableRegistration: utils.Ptr(true)})).To(Succeed())
			})

			AfterAll(func() {
				if github != nil {
					github.server.Close()
				}

				if h != nil {
					h.Close()
				}
			})

			loginThroughGithub := func(user stubGithubUser) (*http.Response, string) {
				github.actAs(user)
				loginToken := beginLogin(h)
				browser := newBrowser()
				_, body := startIdentityProviderLogin(browser, h, loginToken, "github")

				resp, err := browser.Get(body["authorizationUrl"].(string))
				Expect(err).ToNot(HaveOccurred())
				defer resp.Body.Close() //nolint:errcheck
				Expect(resp.StatusCode).To(Equal(http.StatusFound))
				callback, err := url.Parse(resp.Header.Get("Location"))
				Expect(err).ToNot(HaveOccurred())

				return callbackAtKeyline(browser, h, "github", callback.Query()), loginToken
			}

			It("logs in a github user with a public email", func() {
				resp, loginToken := loginThroughGithub(stubGithubUser{Id: 583231, Login: "octocat", Name: "The Octocat", Email: utils.Ptr("octocat@example.com"), Emails: []stubGithubEmail{
					{Email: "octocat@example.com", Primary: true, Verified: true},
				}})

				expectLoginRedirect(resp, loginToken, "")
				Expect(loginState(h, loginToken)["step"]).To(Equal("finish"))
				users := dexUsersNamed(h, "octocat")
				Expect(users).To(HaveLen(1))
				Expect(users[0].PrimaryEmail).To(Equal("octocat@example.com"))
				Expect(users[0].DisplayName).To(Equal("The Octocat"))
			})

			It("finds a private email on the emails endpoint", func() {
				resp, loginToken := loginThroughGithub(stubGithubUser{Id: 7, Login: "hubot", Name: "Hubot", Email: nil, Emails: []stubGithubEmail{
					{Email: "old@example.com", Primary: false, Verified: true},
					{Email: "hubot@example.com", Primary: true, Verified: true},
				}})

				expectLoginRedirect(resp, loginToken, "")
				users := dexUsersNamed(h, "hubot")
				Expect(users).To(HaveLen(1))
				Expect(users[0].PrimaryEmail).To(Equal("hubot@example.com"))
			})

			It("logs the same github user in again by id even after a rename", func() {
				resp, loginToken := loginThroughGithub(stubGithubUser{Id: 583231, Login: "octocat-renamed", Name: "The Octocat", Email: utils.Ptr("octocat@example.com"), Emails: []stubGithubEmail{
					{Email: "octocat@example.com", Primary: true, Verified: true},
				}})

				expectLoginRedirect(resp, loginToken, "")
				Expect(dexUsersNamed(h, "octocat")).To(HaveLen(1))
				Expect(dexUsersNamed(h, "octocat-renamed")).To(BeEmpty())
			})

			It("ignores the profile email and trusts only the emails endpoint", func() {
				resp, loginToken := loginThroughGithub(stubGithubUser{Id: 11, Login: "braggart", Name: "Braggart", Email: utils.Ptr("ceo@example.com"), Emails: []stubGithubEmail{
					{Email: "braggart@example.com", Primary: true, Verified: true},
				}})

				expectLoginRedirect(resp, loginToken, "")
				users := dexUsersNamed(h, "braggart")
				Expect(users).To(HaveLen(1))
				Expect(users[0].PrimaryEmail).To(Equal("braggart@example.com"))
			})

			It("refuses a github user whose primary email is not verified", func() {
				resp, loginToken := loginThroughGithub(stubGithubUser{Id: 8, Login: "shady", Email: nil, Emails: []stubGithubEmail{
					{Email: "shady@example.com", Primary: true, Verified: false},
				}})

				expectLoginRedirect(resp, loginToken, "identity_provider")
				Expect(dexUsersNamed(h, "shady")).To(BeEmpty())
			})

			It("refuses a github user whose login is a taken username", func() {
				resp, loginToken := loginThroughGithub(stubGithubUser{Id: 9, Login: "octocat", Name: "Impostor", Email: utils.Ptr("impostor@example.com"), Emails: []stubGithubEmail{
					{Email: "impostor@example.com", Primary: true, Verified: true},
				}})

				expectLoginRedirect(resp, loginToken, "identity_provider")
				Expect(dexUsersNamed(h, "octocat")).To(HaveLen(1))
			})

			It("refuses a github user whose email belongs to another user", func() {
				resp, loginToken := loginThroughGithub(stubGithubUser{Id: 10, Login: "sneaky", Name: "Sneaky", Email: utils.Ptr("octocat@example.com"), Emails: []stubGithubEmail{
					{Email: "octocat@example.com", Primary: true, Verified: true},
				}})

				expectLoginRedirect(resp, loginToken, "identity_provider")
				Expect(dexUsersNamed(h, "sneaky")).To(BeEmpty())
			})
		})
	}
}
