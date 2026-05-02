//go:build e2e

package e2e

import (
	"context"
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	devAuthConfClient = "test-device-conf-app"
	devAuthPubClient  = "test-device-pub-app"
	devAuthUser       = "test-device-auth-user"
	devAuthPassword   = "test-device-auth-password-1"
)

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Device flow client authentication ["+backend.name+"]", Ordered, func() {
			var h *harness
			var confSecret string

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)
				var err error
				confSecret, err = setupDeviceFlowClientAuthFixtures(h.Scope())
				Expect(err).ToNot(HaveOccurred())
				Expect(confSecret).ToNot(BeEmpty())
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			postForm := func(path string, form url.Values) (int, map[string]any) {
				req, err := http.NewRequest(http.MethodPost, h.ApiUrl()+path, strings.NewReader(form.Encode()))
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				resp, err := http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				defer resp.Body.Close() //nolint:errcheck

				body, _ := io.ReadAll(resp.Body)
				var parsed map[string]any
				_ = json.Unmarshal(body, &parsed)
				return resp.StatusCode, parsed
			}

			postDevice := func(form url.Values) (int, map[string]any) {
				return postForm("/oidc/test-vs/device", form)
			}

			postToken := func(form url.Values) (int, map[string]any) {
				return postForm("/oidc/test-vs/token", form)
			}

			beginDeviceFlowAuthorized := func(clientId, secret string) (deviceCode, userCode string) {
				form := url.Values{
					"client_id": {clientId},
					"scope":     {"openid"},
				}
				if secret != "" {
					form.Set("client_secret", secret)
				}
				status, body := postDevice(form)
				Expect(status).To(Equal(http.StatusOK), "begin /device for %s should succeed: %v", clientId, body)
				dc, _ := body["device_code"].(string)
				uc, _ := body["user_code"].(string)
				Expect(dc).ToNot(BeEmpty())
				Expect(uc).ToNot(BeEmpty())

				loginToken, err := h.Client().Oidc().PostActivate(h.Ctx(), uc)
				Expect(err).ToNot(HaveOccurred())
				err = h.Client().Oidc().VerifyPassword(h.Ctx(), loginToken, devAuthUser, devAuthPassword)
				Expect(err).ToNot(HaveOccurred())
				err = h.Client().Oidc().FinishLogin(h.Ctx(), loginToken)
				Expect(err).ToNot(HaveOccurred())

				return dc, uc
			}

			Describe("/device endpoint", func() {
				It("rejects confidential client without a client_secret", func() {
					status, body := postDevice(url.Values{
						"client_id": {devAuthConfClient},
						"scope":     {"openid"},
					})
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})

				It("rejects confidential client with a wrong client_secret", func() {
					status, body := postDevice(url.Values{
						"client_id":     {devAuthConfClient},
						"client_secret": {"not-the-real-secret"},
						"scope":         {"openid"},
					})
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})

				It("rejects public client when a client_secret is presented", func() {
					status, body := postDevice(url.Values{
						"client_id":     {devAuthPubClient},
						"client_secret": {"any-value"},
						"scope":         {"openid"},
					})
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})

				It("accepts confidential client with the correct client_secret", func() {
					status, body := postDevice(url.Values{
						"client_id":     {devAuthConfClient},
						"client_secret": {confSecret},
						"scope":         {"openid"},
					})
					Expect(status).To(Equal(http.StatusOK))
					Expect(body["device_code"]).ToNot(BeEmpty())
					Expect(body["user_code"]).ToNot(BeEmpty())
				})

				It("accepts public client without a client_secret", func() {
					status, body := postDevice(url.Values{
						"client_id": {devAuthPubClient},
						"scope":     {"openid"},
					})
					Expect(status).To(Equal(http.StatusOK))
					Expect(body["device_code"]).ToNot(BeEmpty())
					Expect(body["user_code"]).ToNot(BeEmpty())
				})
			})

			Describe("/token (device_code grant)", func() {
				It("rejects confidential client redemption without client_secret", func() {
					deviceCode, _ := beginDeviceFlowAuthorized(devAuthConfClient, confSecret)

					status, body := postToken(url.Values{
						"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
						"client_id":   {devAuthConfClient},
						"device_code": {deviceCode},
					})
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})

				It("rejects confidential client redemption with wrong client_secret", func() {
					deviceCode, _ := beginDeviceFlowAuthorized(devAuthConfClient, confSecret)

					status, body := postToken(url.Values{
						"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
						"client_id":     {devAuthConfClient},
						"client_secret": {"wrong-secret"},
						"device_code":   {deviceCode},
					})
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})

				It("rejects public client redemption that includes a client_secret", func() {
					deviceCode, _ := beginDeviceFlowAuthorized(devAuthPubClient, "")

					status, body := postToken(url.Values{
						"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
						"client_id":     {devAuthPubClient},
						"client_secret": {"unexpected"},
						"device_code":   {deviceCode},
					})
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_client"))
				})

				It("accepts confidential client redemption with the correct client_secret", func() {
					deviceCode, _ := beginDeviceFlowAuthorized(devAuthConfClient, confSecret)

					status, body := postToken(url.Values{
						"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
						"client_id":     {devAuthConfClient},
						"client_secret": {confSecret},
						"device_code":   {deviceCode},
					})
					Expect(status).To(Equal(http.StatusOK))
					Expect(body["access_token"]).ToNot(BeNil())
					Expect(body["id_token"]).ToNot(BeNil())
				})

				It("rejects redemption with a different client_id than the device_code was issued to", func() {
					// device_code issued to the confidential client; attacker tries
					// to redeem it as the public client (which would pass the
					// no-secret authentication check). The code/client binding
					// must reject this.
					deviceCode, _ := beginDeviceFlowAuthorized(devAuthConfClient, confSecret)

					status, body := postToken(url.Values{
						"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
						"client_id":   {devAuthPubClient},
						"device_code": {deviceCode},
					})
					Expect(status).To(Equal(http.StatusBadRequest))
					Expect(body["error"]).To(Equal("invalid_grant"))
				})
			})
		})
	}
}

func setupDeviceFlowClientAuthFixtures(scope *ioc.DependencyProvider) (string, error) {
	subscope := scope.NewScope()
	defer subscope.Close()

	ctx := context.Background()
	ctx = middlewares.ContextWithScope(ctx, subscope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	m := ioc.GetDependency[mediatr.Mediator](subscope)
	dbContext := ioc.GetDependency[database.Context](subscope)

	_, err := mediatr.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: "test-vs",
		Slug:              "device-flow-clientauth-project",
		Name:              "Device Flow Client Auth Project",
	})
	if err != nil {
		return "", fmt.Errorf("creating project: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return "", fmt.Errorf("saving project: %w", err)
	}

	confResp, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      "test-vs",
		ProjectSlug:            "device-flow-clientauth-project",
		Name:                   devAuthConfClient,
		DisplayName:            "Test Device Confidential App",
		Type:                   repositories.ApplicationTypeConfidential,
		RedirectUris:           []string{"http://localhost:9999/callback"},
		PostLogoutRedirectUris: []string{},
		DeviceFlowEnabled:      true,
	})
	if err != nil {
		return "", fmt.Errorf("creating confidential application: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return "", fmt.Errorf("saving confidential application: %w", err)
	}
	if confResp.Secret == nil {
		return "", fmt.Errorf("confidential client created without a secret")
	}
	confSecret := *confResp.Secret

	_, err = mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      "test-vs",
		ProjectSlug:            "device-flow-clientauth-project",
		Name:                   devAuthPubClient,
		DisplayName:            "Test Device Public App",
		Type:                   repositories.ApplicationTypePublic,
		RedirectUris:           []string{"http://localhost:9999/callback"},
		PostLogoutRedirectUris: []string{},
		DeviceFlowEnabled:      true,
	})
	if err != nil {
		return "", fmt.Errorf("creating public application: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return "", fmt.Errorf("saving public application: %w", err)
	}

	userResp, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: "test-vs",
		DisplayName:       "Test Device Auth User",
		Username:          devAuthUser,
		Email:             devAuthUser + "@test.local",
		EmailVerified:     true,
	})
	if err != nil {
		return "", fmt.Errorf("creating user: %w", err)
	}
	if err := dbContext.SaveChanges(ctx); err != nil {
		return "", fmt.Errorf("saving user: %w", err)
	}

	passwordCred := repositories.NewCredential(userResp.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: utils.HashPassword(devAuthPassword),
		Temporary:      false,
	})
	dbContext.Credentials().Insert(passwordCred)
	if err := dbContext.SaveChanges(ctx); err != nil {
		return "", fmt.Errorf("saving password credential: %w", err)
	}

	return confSecret, nil
}
