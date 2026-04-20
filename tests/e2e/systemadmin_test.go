//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/utils"
	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	systemAdminServiceUserUsername   = "test-system-admin-user"
	systemAdminServiceUserKid        = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	systemAdminServiceUserPublicKey  = "-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEAX3J/Yilw4CTcsOVW0BBasQwY9wuYwcJZkJliqAhNa5s=\n-----END PUBLIC KEY-----\n"
	systemAdminServiceUserPrivateKey = "-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEIJkHOgIL4pqTWGAxhEX+VxenOkoevvegT1LkKTJAG/cu\n-----END PRIVATE KEY-----\n"

	unprivilegedServiceUserUsername   = "test-unprivileged-user"
	unprivilegedServiceUserKid        = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	unprivilegedServiceUserPublicKey  = "-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEAxtYtnDiK1itrLhvF/x8gOqeKMVk2wrCKhGID2WzAjA4=\n-----END PUBLIC KEY-----\n"
	unprivilegedServiceUserPrivateKey = "-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEII0CRu97ywJ0abQDlka2YwfCxXg6x59EQ8HEnKap80m0\n-----END PRIVATE KEY-----\n"

	secondVirtualServerName = "second-vs"
	crossVsTestProjectSlug  = "cross-vs-test-project"
)

func setupSystemAdminFixtures(h *harness) {
	scope := h.Scope().NewScope()
	defer scope.Close()

	ctx := middlewares.ContextWithScope(context.Background(), scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
	m := ioc.GetDependency[mediatr.Mediator](scope)
	dbContext := ioc.GetDependency[database.Context](scope)

	// Create the system-admin role in the initial VS's system project.
	// The system project always has slug "system", so the token claim becomes "system:system-admin".
	createRoleResp, err := mediatr.Send[*commands.CreateRoleResponse](ctx, m, commands.CreateRole{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       "system",
		Name:              commands.SystemAdminRoleName,
		Description:       "System administrator role",
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	serviceUserResp, err := mediatr.Send[*commands.CreateServiceUserResponse](ctx, m, commands.CreateServiceUser{
		VirtualServerName: h.VirtualServer(),
		Username:          systemAdminServiceUserUsername,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	_, err = mediatr.Send[*commands.AssignRoleToUserResponse](ctx, m, commands.AssignRoleToUser{
		VirtualServerName: h.VirtualServer(),
		ProjectSlug:       "system",
		UserId:            serviceUserResp.Id,
		RoleId:            createRoleResp.Id,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	_, err = mediatr.Send[*commands.AssociateServiceUserPublicKeyResponse](ctx, m, commands.AssociateServiceUserPublicKey{
		VirtualServerName: h.VirtualServer(),
		ServiceUserId:     serviceUserResp.Id,
		PublicKey:         systemAdminServiceUserPublicKey,
		Kid:               utils.Ptr(systemAdminServiceUserKid),
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	// Create a second VS to use as the cross-VS target.
	_, err = mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:                    secondVirtualServerName,
		DisplayName:             "Second Virtual Server",
		PrimarySigningAlgorithm: config.SigningAlgorithmEdDSA,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	// Create a project in the second VS so the app-creation tests have a project to target.
	_, err = mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: secondVirtualServerName,
		Slug:              crossVsTestProjectSlug,
		Name:              "Cross-VS Test Project",
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	// Create a service user with no roles — used for the negative test.
	unprivUserResp, err := mediatr.Send[*commands.CreateServiceUserResponse](ctx, m, commands.CreateServiceUser{
		VirtualServerName: h.VirtualServer(),
		Username:          unprivilegedServiceUserUsername,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())

	_, err = mediatr.Send[*commands.AssociateServiceUserPublicKeyResponse](ctx, m, commands.AssociateServiceUserPublicKey{
		VirtualServerName: h.VirtualServer(),
		ServiceUserId:     unprivUserResp.Id,
		PublicKey:         unprivilegedServiceUserPublicKey,
		Kid:               utils.Ptr(unprivilegedServiceUserKid),
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(dbContext.SaveChanges(ctx)).To(Succeed())
}

// acquireTokenForServiceUser exchanges a signed service-user JWT for an access token
// from the given VS's OIDC endpoint.
func acquireTokenForServiceUser(h *harness, username, kid, privateKeyPem string) string {
	return acquireTokenForServiceUserOnVS(h, h.VirtualServer(), commands.AdminApplicationName, username, kid, privateKeyPem)
}

func acquireTokenForServiceUserOnVS(h *harness, vsName, appName, username, kid, privateKeyPem string) string {
	block, _ := pem.Decode([]byte(privateKeyPem))
	Expect(block).ToNot(BeNil())

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	Expect(err).ToNot(HaveOccurred())

	claims := jwt.MapClaims{
		"aud":    appName,
		"iss":    username,
		"sub":    username,
		"scopes": "openid profile email",
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	jwtToken.Header["kid"] = kid
	signedJWT, err := jwtToken.SignedString(key)
	Expect(err).ToNot(HaveOccurred())

	resp, err := http.PostForm(
		fmt.Sprintf("%s/oidc/%s/token", h.ApiUrl(), vsName),
		url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":      {signedJWT},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		},
	)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var tokenResp map[string]any
	Expect(json.NewDecoder(resp.Body).Decode(&tokenResp)).To(Succeed())

	token, ok := tokenResp["access_token"].(string)
	Expect(ok).To(BeTrue())
	Expect(token).ToNot(BeEmpty())
	return token
}

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("System admin cross-VS auth ["+backend.name+"]", Ordered, func() {
			var h *harness
			var systemAdminToken string

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				setupSystemAdminFixtures(h)
				systemAdminToken = acquireTokenForServiceUser(h,
					systemAdminServiceUserUsername,
					systemAdminServiceUserKid,
					systemAdminServiceUserPrivateKey,
				)
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("can create a virtual server using the initial-VS token", func() {
				req, err := http.NewRequest(http.MethodPost,
					fmt.Sprintf("%s/api/virtual-servers", h.ApiUrl()),
					nil,
				)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", "Bearer "+systemAdminToken)
				req.Header.Set("Content-Type", "application/json")

				// A missing/invalid body returns 400, not 401 — proof the token was accepted.
				resp, err := http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).ToNot(Equal(http.StatusUnauthorized))
			})

			It("is authenticated (not a token-validation failure) on a cross-VS endpoint", func() {
				// The token was issued by the initial VS ("test-vs") but the request targets
				// "second-vs". Without the fix the middleware rejects it with a token-parsing
				// error; with the fix it falls back to the initial VS key, authenticates the
				// user, and lets the policy layer decide (permission denied).
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("%s/api/virtual-servers/%s", h.ApiUrl(), secondVirtualServerName),
					nil,
				)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", "Bearer "+systemAdminToken)

				resp, err := http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())

				body, err := io.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				// Token validation failure produces "parsing token" in the body.
				// Policy denial produces "request not allowed". We want the latter.
				Expect(string(body)).ToNot(ContainSubstring("parsing token"),
					"token from initial VS should be accepted on cross-VS endpoints")
			})

			It("admin from initial VS can create an application in the new VS", func() {
				// The existing service user has the admin (VirtualServerAdmin) role,
				// which grants ApplicationCreate. Their token is issued by the initial VS
				// ("test-vs") but must be honored on "second-vs" via the cross-VS fallback.
				adminToken := acquireTokenForServiceUser(h,
					serviceUserUsername,
					serviceUserKid,
					serviceUserPrivateKey,
				)

				body, err := json.Marshal(map[string]any{
					"name":         "cross-vs-app",
					"displayName":  "Cross-VS App",
					"redirectUris": []string{"http://localhost/callback"},
					"type":         "public",
				})
				Expect(err).ToNot(HaveOccurred())

				req, err := http.NewRequest(http.MethodPost,
					fmt.Sprintf("%s/api/virtual-servers/%s/projects/%s/applications",
						h.ApiUrl(), secondVirtualServerName, crossVsTestProjectSlug),
					bytes.NewReader(body),
				)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", "Bearer "+adminToken)
				req.Header.Set("Content-Type", "application/json")

				resp, err := http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())

				respBody, err := io.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusCreated),
					"admin token from initial VS should be able to create apps in a new VS, got: %s", respBody)
			})

			It("user without the admin role cannot create an application in the new VS", func() {
				// The unprivileged service user has no roles, so their token carries no
				// system: permissions. Even though the cross-VS fallback authenticates
				// the token, the ApplicationCreate permission check must deny the request.
				unprivToken := acquireTokenForServiceUser(h,
					unprivilegedServiceUserUsername,
					unprivilegedServiceUserKid,
					unprivilegedServiceUserPrivateKey,
				)

				body, err := json.Marshal(map[string]any{
					"name":         "should-not-be-created",
					"displayName":  "Should Not Be Created",
					"redirectUris": []string{"http://localhost/callback"},
					"type":         "public",
				})
				Expect(err).ToNot(HaveOccurred())

				req, err := http.NewRequest(http.MethodPost,
					fmt.Sprintf("%s/api/virtual-servers/%s/projects/%s/applications",
						h.ApiUrl(), secondVirtualServerName, crossVsTestProjectSlug),
					bytes.NewReader(body),
				)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", "Bearer "+unprivToken)
				req.Header.Set("Content-Type", "application/json")

				resp, err := http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})
	}
}
