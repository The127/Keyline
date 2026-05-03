//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	passkeyAttackerVS    = "passkey-attacker-vs"
	passkeyTargetUser    = "passkey-target-user"
	passkeyAttackerUser  = "passkey-attacker-user"
	passkeyPassword      = "passkey-test-password-1"
	passkeyTargetApp     = "passkey-target-app"
	passkeyTargetURI     = "http://localhost:9100/passkey-callback"
	passkeyPkceVerifier  = "passkey-cross-tenant-verifier-padding-padding-12"
	cosePublicKeyEd25519 = -8
)

// passkeyTestKey holds an ed25519 keypair plus the credential id we wire
// into the credentials table for one user. Used to drive
// /passkey/start + /passkey/finish from the test as if a real
// authenticator signed the assertion.
type passkeyTestKey struct {
	credentialID string
	publicKey    ed25519.PublicKey
	privateKey   ed25519.PrivateKey
}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("FinishPasskeyLogin cross-tenant lookup ["+backend.name+"]", Ordered, func() {
			var h *harness
			var attackerKey *passkeyTestKey
			var targetUserKey *passkeyTestKey

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				var err error
				attackerKey, targetUserKey, err = setupPasskeyCrossTenantFixtures(h)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			// mintLoginToken kicks off /authorize on `vs` and returns the
			// freshly-minted loginToken (the value carried in the redirect
			// to /login?token=<...>). The PKCE policy at /authorize is
			// mandatory, so a code_challenge is sent even though we never
			// redeem the code.
			mintLoginToken := func(vs string) string {
				httpClient := &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
				challenge := authCodePkceChallenge(passkeyPkceVerifier)
				url := fmt.Sprintf(
					"%s/oidc/%s/authorize?response_type=code&client_id=%s&"+
						"redirect_uri=%s&scope=openid&state=s&nonce=n&"+
						"code_challenge=%s&code_challenge_method=S256",
					h.ApiUrl(), vs, passkeyTargetApp, passkeyTargetURI, challenge,
				)
				resp, err := httpClient.Get(url)
				Expect(err).ToNot(HaveOccurred())
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusFound))
				loc := resp.Header.Get("Location")
				idx := strings.Index(loc, "token=")
				Expect(idx).ToNot(Equal(-1), "no login token in /authorize redirect: %s", loc)
				token := loc[idx+len("token="):]
				if amp := strings.Index(token, "&"); amp != -1 {
					token = token[:amp]
				}
				return token
			}

			// passkeyFinish drives /passkey/start, signs the challenge with
			// `key`, and POSTs to /passkey/finish. Returns the response so
			// callers can assert on status / body.
			passkeyFinish := func(loginToken string, key *passkeyTestKey) *http.Response {
				startResp, err := http.Post(
					fmt.Sprintf("%s/logins/%s/passkey/start", h.ApiUrl(), loginToken),
					"", nil,
				)
				Expect(err).ToNot(HaveOccurred())
				defer func() { _ = startResp.Body.Close() }()
				Expect(startResp.StatusCode).To(Equal(http.StatusOK))

				var startBody struct {
					Id        uuid.UUID `json:"Id"`
					Challenge string    `json:"Challenge"`
				}
				Expect(json.NewDecoder(startResp.Body).Decode(&startBody)).To(Succeed())

				challengeBytes, err := base64.StdEncoding.DecodeString(startBody.Challenge)
				Expect(err).ToNot(HaveOccurred())

				clientData := map[string]string{
					"type":      "webauthn.get",
					"challenge": base64.RawURLEncoding.EncodeToString(challengeBytes),
					"origin":    "https://test.invalid",
				}
				clientDataBytes, err := json.Marshal(clientData)
				Expect(err).ToNot(HaveOccurred())

				authData := bytes.Repeat([]byte{0}, 37)
				cdHash := sha256.Sum256(clientDataBytes)
				signed := append(append([]byte{}, authData...), cdHash[:]...)
				signature := ed25519.Sign(key.privateKey, signed)

				body := map[string]any{
					"id": startBody.Id,
					"webauthnResponse": map[string]any{
						"id":    key.credentialID,
						"rawId": key.credentialID,
						"response": map[string]any{
							"clientDataJSON":    base64.StdEncoding.EncodeToString(clientDataBytes),
							"authenticatorData": base64.RawURLEncoding.EncodeToString(authData),
							"signature":         base64.RawURLEncoding.EncodeToString(signature),
							"userHandle":        "",
						},
						"authenticatorAttachment": "cross-platform",
						"type":                    "public-key",
					},
				}
				bodyBytes, err := json.Marshal(body)
				Expect(err).ToNot(HaveOccurred())

				resp, err := http.Post(
					fmt.Sprintf("%s/logins/%s/passkey/finish", h.ApiUrl(), loginToken),
					"application/json",
					bytes.NewReader(bodyBytes),
				)
				Expect(err).ToNot(HaveOccurred())
				return resp
			}

			Describe("POST /logins/{loginToken}/passkey/finish", func() {
				It("accepts a webauthn credential whose owner is in the same VS as the loginToken (happy path)", func() {
					loginToken := mintLoginToken(h.VirtualServer())
					resp := passkeyFinish(loginToken, targetUserKey)
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(http.StatusNoContent),
						"same-VS passkey login must succeed")
				})

				It("rejects a webauthn credential whose owner is in another VS (REGRESSION: cross-tenant lookup)", func() {
					// attackerKey is registered against passkeyAttackerUser
					// in passkey-attacker-vs. Used here against test-vs's
					// loginToken. Pre-fix, the unscoped credential lookup
					// returned the attacker's row, the signature verified
					// (it's their key), and loginInfo.UserId was set to a
					// user from another VS. Post-fix, the new gate compares
					// credential.UserId() against loginInfo.VirtualServerId
					// and rejects with Unauthorized.
					loginToken := mintLoginToken(h.VirtualServer())
					resp := passkeyFinish(loginToken, attackerKey)
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized),
						"cross-VS passkey login must be rejected")
				})

				It("rejects a webauthn credential whose RawId does not exist anywhere", func() {
					loginToken := mintLoginToken(h.VirtualServer())
					bogus := &passkeyTestKey{
						credentialID: uuid.NewString(),
						publicKey:    attackerKey.publicKey,
						privateKey:   attackerKey.privateKey,
					}
					resp := passkeyFinish(loginToken, bogus)
					defer resp.Body.Close()
					Expect(resp.StatusCode).ToNot(Equal(http.StatusNoContent),
						"unknown credential id must not authenticate")
				})

				It("accepts the attacker's own credential against the attacker VS's loginToken (control: same-VS variant)", func() {
					// Inverse of the cross-tenant case: the attacker user
					// driving the attacker VS's own login flow with the
					// attacker's own credential must still work. This
					// guards against an over-broad fix that would also
					// block the legitimate same-VS case.
					loginToken := mintLoginToken(passkeyAttackerVS)
					resp := passkeyFinish(loginToken, attackerKey)
					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(http.StatusNoContent),
						"attacker's credential must still authenticate the attacker in their own VS")
				})
			})
		})
	}
}

// setupPasskeyCrossTenantFixtures seeds two VSes -- the harness default
// "test-vs" plus a second "passkey-attacker-vs" -- each with a
// webauthn-enabled user and a public OIDC client. The attacker user's
// credential is the one the cross-tenant exploit attempts to replay
// against the target VS's loginToken.
func setupPasskeyCrossTenantFixtures(h *harness) (*passkeyTestKey, *passkeyTestKey, error) {
	scope := h.Scope().NewScope()
	defer scope.Close()

	ctx := middlewares.ContextWithScope(context.Background(), scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
	m := ioc.GetDependency[mediatr.Mediator](scope)
	dbContext := ioc.GetDependency[database.Context](scope)

	// --- target VS (the harness's existing test-vs) ---
	if _, err := mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: h.VirtualServer(),
		Slug:              "passkey-project",
		Name:              "Passkey Project",
	}); err != nil {
		return nil, nil, fmt.Errorf("target project: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return nil, nil, err
	}
	if _, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       "passkey-project",
		Name:              passkeyTargetApp,
		DisplayName:       "Passkey Target App",
		Type:              repositories.ApplicationTypePublic,
		RedirectUris:      []string{passkeyTargetURI},
	}); err != nil {
		return nil, nil, fmt.Errorf("target app: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return nil, nil, err
	}

	targetUserResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: h.VirtualServer(),
		DisplayName:       "Passkey Target User",
		Username:          passkeyTargetUser,
		Email:             passkeyTargetUser + "@test.local",
		EmailVerified:     true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("target user: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return nil, nil, err
	}

	targetKey, err := generatePasskeyTestKey()
	if err != nil {
		return nil, nil, err
	}
	dbContext.Credentials().Insert(repositories.NewCredential(
		targetUserResp.Id,
		&repositories.CredentialWebauthnDetails{
			CredentialId:       targetKey.credentialID,
			PublicKeyAlgorithm: cosePublicKeyEd25519,
			PublicKey:          targetKey.publicKeyDER(),
		},
	))
	if err := dbContext.SaveChanges(ctx); err != nil {
		return nil, nil, err
	}

	// --- attacker VS (separate VS, separate user, separate credential) ---
	if _, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:                    passkeyAttackerVS,
		DisplayName:             "Passkey Attacker VS",
		PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
	}); err != nil {
		return nil, nil, fmt.Errorf("attacker VS: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return nil, nil, err
	}
	if _, err := mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: passkeyAttackerVS,
		Slug:              "passkey-project",
		Name:              "Attacker Project",
	}); err != nil {
		return nil, nil, fmt.Errorf("attacker project: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return nil, nil, err
	}
	// The attacker VS needs a same-named app so the same /authorize
	// payload (used by mintLoginToken) works against either VS. This
	// mirrors the realistic case where common app names collide across
	// tenants.
	if _, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName: passkeyAttackerVS,
		ProjectSlug:       "passkey-project",
		Name:              passkeyTargetApp,
		DisplayName:       "Attacker Mirror App",
		Type:              repositories.ApplicationTypePublic,
		RedirectUris:      []string{passkeyTargetURI},
	}); err != nil {
		return nil, nil, fmt.Errorf("attacker app: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return nil, nil, err
	}

	attackerUserResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: passkeyAttackerVS,
		DisplayName:       "Passkey Attacker User",
		Username:          passkeyAttackerUser,
		Email:             passkeyAttackerUser + "@attacker.local",
		EmailVerified:     true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("attacker user: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return nil, nil, err
	}

	attackerKey, err := generatePasskeyTestKey()
	if err != nil {
		return nil, nil, err
	}
	dbContext.Credentials().Insert(repositories.NewCredential(
		attackerUserResp.Id,
		&repositories.CredentialWebauthnDetails{
			CredentialId:       attackerKey.credentialID,
			PublicKeyAlgorithm: cosePublicKeyEd25519,
			PublicKey:          attackerKey.publicKeyDER(),
		},
	))
	if err := dbContext.SaveChanges(ctx); err != nil {
		return nil, nil, err
	}

	return attackerKey, targetKey, nil
}

func generatePasskeyTestKey() (*passkeyTestKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	return &passkeyTestKey{
		credentialID: uuid.NewString(),
		publicKey:    pub,
		privateKey:   priv,
	}, nil
}

func (k *passkeyTestKey) publicKeyDER() []byte {
	der, err := x509.MarshalPKIXPublicKey(k.publicKey)
	if err != nil {
		panic(fmt.Errorf("marshal public key: %w", err))
	}
	return der
}
