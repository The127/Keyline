//go:build e2e

package e2e

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	pkJwtProjectSlug   = "pk-jwt"
	pkJwtAppName       = "pk-jwt-app"
	pkJwtSecretAppName = "pk-jwt-secret-app"
	pkJwtRedirect      = "http://localhost:9300/callback"
	pkJwtKid           = "pk-jwt-key-1"

	clientAssertionTypeJwtBearer = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
)

func setupPrivateKeyJwtFixtures(h *harness) uuid.UUID {
	scope := h.Scope().NewScope()
	defer utils.PanicOnError(scope.Close, "closing scope")
	ctx := middlewares.ContextWithScope(context.Background(), scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
	m := ioc.GetDependency[mediatr.Mediator](scope)
	dbContext := ioc.GetDependency[database.Context](scope)

	_, err := mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: h.VirtualServer(),
		Slug:              pkJwtProjectSlug,
		Name:              "Private Key JWT",
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	keyApp, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:       h.VirtualServer(),
		ProjectSlug:             pkJwtProjectSlug,
		Name:                    pkJwtAppName,
		DisplayName:             "Private Key JWT App",
		Type:                    repositories.ApplicationTypeConfidential,
		RedirectUris:            []string{pkJwtRedirect},
		PostLogoutRedirectUris:  []string{},
		AccessTokenHeaderType:   "at+jwt",
		DeviceFlowEnabled:       true,
		TokenEndpointAuthMethod: utils.Ptr(repositories.TokenEndpointAuthMethodPrivateKeyJwt),
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	_, err = mediatr.Send[*commands.AddApplicationKeyResponse](ctx, m, commands.AddApplicationKey{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       pkJwtProjectSlug,
		ApplicationId:     keyApp.Id,
		Kid:               utils.Ptr(pkJwtKid),
		PublicKey:         serviceUserPublicKey,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	_, err = mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      h.VirtualServer(),
		ProjectSlug:            pkJwtProjectSlug,
		Name:                   pkJwtSecretAppName,
		DisplayName:            "Private Key JWT Secret App",
		Type:                   repositories.ApplicationTypeConfidential,
		RedirectUris:           []string{pkJwtRedirect},
		PostLogoutRedirectUris: []string{},
		AccessTokenHeaderType:  "at+jwt",
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	user, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: h.VirtualServer(),
		DisplayName:       "Private Key JWT User",
		Username:          authCodeUserName,
		Email:             authCodeUserName + "@test.local",
		EmailVerified:     true,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	dbContext.Credentials().Insert(repositories.NewCredential(user.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: utils.HashPassword(authCodeUserPassword),
		Temporary:      false,
	}))
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	return keyApp.Id
}

func signClientAssertion(privateKeyPem string, kid string, claims jwt.MapClaims) string {
	block, _ := pem.Decode([]byte(privateKeyPem))
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	Expect(err).ToNot(HaveOccurred())

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	if kid != "" {
		token.Header["kid"] = kid
	}
	signed, err := token.SignedString(key)
	Expect(err).ToNot(HaveOccurred())
	return signed
}

func clientAssertionClaims(h *harness, now time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"iss": pkJwtAppName,
		"sub": pkJwtAppName,
		"aud": fmt.Sprintf("%s/oidc/%s", h.ApiUrl(), h.VirtualServer()),
		"iat": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
		"jti": uuid.New().String(),
	}
}

func clientAssertion(h *harness, now time.Time, mutate func(claims jwt.MapClaims)) string {
	claims := clientAssertionClaims(h, now)
	if mutate != nil {
		mutate(claims)
	}
	return signClientAssertion(serviceUserPrivateKey, pkJwtKid, claims)
}

func withClientAssertion(form url.Values, assertion string) url.Values {
	form.Set("client_assertion_type", clientAssertionTypeJwtBearer)
	form.Set("client_assertion", assertion)
	return form
}

func expectInvalidClient(status int, body map[string]any) {
	ExpectWithOffset(1, status).To(Equal(http.StatusBadRequest), fmt.Sprintf("body: %v", body))
	ExpectWithOffset(1, body["error"]).To(Equal("invalid_client"), fmt.Sprintf("body: %v", body))
	ExpectWithOffset(1, body).ToNot(HaveKey("access_token"))
}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Client authentication with private_key_jwt ["+backend.name+"]", Ordered, func() {
			var h *harness
			var now time.Time
			var keyAppId uuid.UUID
			var refreshToken string

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				now = time.Now().Truncate(time.Second)
				h.SetTime(now)
				keyAppId = setupPrivateKeyJwtFixtures(h)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			redeemForm := func() url.Values {
				code, err := authCodeFlow(h.ApiUrl(), pkJwtAppName, pkJwtRedirect, authCodePkceChallenge(authCodePkceVerifier))
				Expect(err).ToNot(HaveOccurred())
				return url.Values{
					"grant_type":    {"authorization_code"},
					"code":          {code},
					"redirect_uri":  {pkJwtRedirect},
					"code_verifier": {authCodePkceVerifier},
				}
			}

			refreshForm := func() url.Values {
				return url.Values{
					"grant_type":    {"refresh_token"},
					"refresh_token": {refreshToken},
				}
			}

			refreshWith := func(form url.Values) {
				status, body, err := postToken(h.ApiUrl(), form)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(http.StatusOK), fmt.Sprintf("body: %v", body))
				Expect(body["access_token"]).ToNot(BeEmpty())
				refreshToken = body["refresh_token"].(string)
			}

			It("advertises private_key_jwt in discovery", func() {
				resp, err := http.Get(fmt.Sprintf("%s/oidc/%s/.well-known/openid-configuration", h.ApiUrl(), h.VirtualServer()))
				Expect(err).ToNot(HaveOccurred())
				defer resp.Body.Close() //nolint:errcheck

				var discovery map[string]any
				Expect(json.NewDecoder(resp.Body).Decode(&discovery)).To(Succeed())
				Expect(discovery["token_endpoint_auth_methods_supported"]).To(ContainElement("private_key_jwt"))
				Expect(discovery["token_endpoint_auth_signing_alg_values_supported"]).To(ConsistOf("RS256", "EdDSA"))
			})

			It("redeems an authorization code with a client assertion and no client_id", func() {
				status, body, err := postToken(h.ApiUrl(), withClientAssertion(redeemForm(), clientAssertion(h, now, nil)))
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(http.StatusOK), fmt.Sprintf("body: %v", body))
				Expect(body["access_token"]).ToNot(BeEmpty())
				Expect(body["id_token"]).ToNot(BeEmpty())
				Expect(body["refresh_token"]).ToNot(BeEmpty())
				refreshToken = body["refresh_token"].(string)
			})

			It("refreshes with a client assertion", func() {
				refreshWith(withClientAssertion(refreshForm(), clientAssertion(h, now, nil)))
			})

			It("accepts a matching client_id next to the assertion", func() {
				form := withClientAssertion(refreshForm(), clientAssertion(h, now, nil))
				form.Set("client_id", pkJwtAppName)
				refreshWith(form)
			})

			It("accepts the token endpoint as audience", func() {
				tokenEndpoint := fmt.Sprintf("%s/oidc/%s/token", h.ApiUrl(), h.VirtualServer())
				refreshWith(withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { c["aud"] = tokenEndpoint })))
			})

			It("runs the device flow with a client assertion and no client_id", func() {
				resp, err := http.PostForm(fmt.Sprintf("%s/oidc/%s/device", h.ApiUrl(), h.VirtualServer()), withClientAssertion(url.Values{"scope": {"openid"}}, clientAssertion(h, now, nil)))
				Expect(err).ToNot(HaveOccurred())
				defer resp.Body.Close() //nolint:errcheck
				var device map[string]any
				Expect(json.NewDecoder(resp.Body).Decode(&device)).To(Succeed())
				Expect(resp.StatusCode).To(Equal(http.StatusOK), fmt.Sprintf("body: %v", device))
				deviceCode := device["device_code"].(string)
				userCode := device["user_code"].(string)

				loginToken, err := h.Client().Oidc().PostActivate(h.Ctx(), userCode)
				Expect(err).ToNot(HaveOccurred())
				Expect(h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, authCodeUserName, authCodeUserPassword)).To(Succeed())
				Expect(h.Client().Oidc().FinishLogin(h.Ctx(), loginToken)).To(Succeed())

				status, body, err := postToken(h.ApiUrl(), withClientAssertion(url.Values{
					"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
					"device_code": {deviceCode},
				}, clientAssertion(h, now, nil)))
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(http.StatusOK), fmt.Sprintf("body: %v", body))
				Expect(body["access_token"]).ToNot(BeEmpty())
			})

			It("refuses the code for a client_secret on a private_key_jwt application", func() {
				form := redeemForm()
				form.Set("client_id", pkJwtAppName)
				form.Set("client_secret", "whatever")
				status, body, err := postToken(h.ApiUrl(), form)
				Expect(err).ToNot(HaveOccurred())
				expectInvalidClient(status, body)
			})

			Describe("refuses a client assertion", func() {
				cases := map[string]func() url.Values{
					"signed with the wrong key": func() url.Values {
						return withClientAssertion(refreshForm(), signClientAssertion(wrongPrivateKey, pkJwtKid, clientAssertionClaims(h, now)))
					},
					"with an unknown kid": func() url.Values {
						return withClientAssertion(refreshForm(), signClientAssertion(serviceUserPrivateKey, "no-such-kid", clientAssertionClaims(h, now)))
					},
					"without a kid header": func() url.Values {
						return withClientAssertion(refreshForm(), signClientAssertion(serviceUserPrivateKey, "", clientAssertionClaims(h, now)))
					},
					"whose iss differs from sub": func() url.Values {
						return withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { c["iss"] = "someone-else" }))
					},
					"naming an unknown client": func() url.Values {
						return withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { c["iss"] = "nobody"; c["sub"] = "nobody" }))
					},
					"naming a client_secret application": func() url.Values {
						return withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { c["iss"] = pkJwtSecretAppName; c["sub"] = pkJwtSecretAppName }))
					},
					"with a client_id that does not match": func() url.Values {
						form := withClientAssertion(refreshForm(), clientAssertion(h, now, nil))
						form.Set("client_id", pkJwtSecretAppName)
						return form
					},
					"combined with a client_secret": func() url.Values {
						form := withClientAssertion(refreshForm(), clientAssertion(h, now, nil))
						form.Set("client_secret", "whatever")
						return form
					},
					"addressed to another audience": func() url.Values {
						return withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { c["aud"] = "https://elsewhere.example" }))
					},
					"without aud": func() url.Values {
						return withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { delete(c, "aud") }))
					},
					"without exp": func() url.Values {
						return withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { delete(c, "exp") }))
					},
					"that has expired": func() url.Values {
						return withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Second).Unix() }))
					},
					"that lives longer than five minutes": func() url.Values {
						return withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { c["exp"] = now.Add(time.Hour).Unix() }))
					},
					"without jti": func() url.Values {
						return withClientAssertion(refreshForm(), clientAssertion(h, now, func(c jwt.MapClaims) { delete(c, "jti") }))
					},
					"with an unsupported assertion type": func() url.Values {
						form := withClientAssertion(refreshForm(), clientAssertion(h, now, nil))
						form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:saml2-bearer")
						return form
					},
					"with an assertion type but no assertion": func() url.Values {
						form := refreshForm()
						form.Set("client_assertion_type", clientAssertionTypeJwtBearer)
						return form
					},
					"that is garbage": func() url.Values {
						return withClientAssertion(refreshForm(), "not.a.jwt")
					},
					"signed with alg none": func() url.Values {
						token := jwt.NewWithClaims(jwt.SigningMethodNone, clientAssertionClaims(h, now))
						token.Header["kid"] = pkJwtKid
						unsigned, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
						Expect(err).ToNot(HaveOccurred())
						return withClientAssertion(refreshForm(), unsigned)
					},
				}
				for name, form := range cases {
					form := form
					It(name, func() {
						status, body, err := postToken(h.ApiUrl(), form())
						Expect(err).ToNot(HaveOccurred())
						expectInvalidClient(status, body)
					})
				}

				It("and the refresh token is still usable afterwards", func() {
					refreshWith(withClientAssertion(refreshForm(), clientAssertion(h, now, nil)))
				})
			})

			It("refuses a replayed assertion and accepts a fresh one", func() {
				assertion := clientAssertion(h, now, nil)
				refreshWith(withClientAssertion(refreshForm(), assertion))

				status, body, err := postToken(h.ApiUrl(), withClientAssertion(refreshForm(), assertion))
				Expect(err).ToNot(HaveOccurred())
				expectInvalidClient(status, body)

				refreshWith(withClientAssertion(refreshForm(), clientAssertion(h, now, nil)))
			})

			It("accepts a replayed jti again once the original assertion has expired", func() {
				assertion := clientAssertion(h, now, func(c jwt.MapClaims) { c["jti"] = "fixed-jti" })
				refreshWith(withClientAssertion(refreshForm(), assertion))

				later := now.Add(2 * time.Minute)
				h.SetTime(later)
				defer h.SetTime(now)

				refreshWith(withClientAssertion(refreshForm(), clientAssertion(h, later, func(c jwt.MapClaims) { c["jti"] = "fixed-jti" })))
			})

			It("accepts a freshly generated key once it is registered", func() {
				publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
				Expect(err).ToNot(HaveOccurred())
				publicDer, err := x509.MarshalPKIXPublicKey(publicKey)
				Expect(err).ToNot(HaveOccurred())
				privateDer, err := x509.MarshalPKCS8PrivateKey(privateKey)
				Expect(err).ToNot(HaveOccurred())
				publicPem := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDer}))
				privatePem := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDer}))

				assertion := signClientAssertion(privatePem, "rotated", clientAssertionClaims(h, now))
				status, body, err := postToken(h.ApiUrl(), withClientAssertion(refreshForm(), assertion))
				Expect(err).ToNot(HaveOccurred())
				expectInvalidClient(status, body)

				withAdminScope(h, func(ctx context.Context, m mediatr.Mediator, db database.Context) {
					_, err := mediatr.Send[*commands.AddApplicationKeyResponse](ctx, m, commands.AddApplicationKey{
						VirtualServerName: h.VirtualServer(),
						ProjectSlug:       pkJwtProjectSlug,
						ApplicationId:     keyAppId,
						Kid:               utils.Ptr("rotated"),
						PublicKey:         publicPem,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(db.SaveChanges(ctx)).To(Succeed())
				})

				refreshWith(withClientAssertion(refreshForm(), signClientAssertion(privatePem, "rotated", clientAssertionClaims(h, now))))
			})

			It("accepts an RSA key signed with RS256", func() {
				privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
				Expect(err).ToNot(HaveOccurred())
				publicDer, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
				Expect(err).ToNot(HaveOccurred())
				publicPem := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDer}))

				withAdminScope(h, func(ctx context.Context, m mediatr.Mediator, db database.Context) {
					_, err := mediatr.Send[*commands.AddApplicationKeyResponse](ctx, m, commands.AddApplicationKey{
						VirtualServerName: h.VirtualServer(),
						ProjectSlug:       pkJwtProjectSlug,
						ApplicationId:     keyAppId,
						Kid:               utils.Ptr("rsa"),
						PublicKey:         publicPem,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(db.SaveChanges(ctx)).To(Succeed())
				})

				token := jwt.NewWithClaims(jwt.SigningMethodRS256, clientAssertionClaims(h, now))
				token.Header["kid"] = "rsa"
				assertion, err := token.SignedString(privateKey)
				Expect(err).ToNot(HaveOccurred())
				refreshWith(withClientAssertion(refreshForm(), assertion))
			})

			It("refuses the assertion once its key is removed", func() {
				withAdminScope(h, func(ctx context.Context, m mediatr.Mediator, db database.Context) {
					_, err := mediatr.Send[*commands.RemoveApplicationKeyResponse](ctx, m, commands.RemoveApplicationKey{
						VirtualServerName: h.VirtualServer(),
						ProjectSlug:       pkJwtProjectSlug,
						ApplicationId:     keyAppId,
						Kid:               pkJwtKid,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(db.SaveChanges(ctx)).To(Succeed())
				})

				status, body, err := postToken(h.ApiUrl(), withClientAssertion(refreshForm(), clientAssertion(h, now, nil)))
				Expect(err).ToNot(HaveOccurred())
				expectInvalidClient(status, body)
			})
		})
	}
}

func withAdminScope(h *harness, f func(ctx context.Context, m mediatr.Mediator, db database.Context)) {
	scope := h.Scope().NewScope()
	defer utils.PanicOnError(scope.Close, "closing scope")
	ctx := middlewares.ContextWithScope(context.Background(), scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
	f(ctx, ioc.GetDependency[mediatr.Mediator](scope), ioc.GetDependency[database.Context](scope))
}
