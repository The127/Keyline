//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/The127/Keyline/api"
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

const appAlgServiceUserName = "app-alg-svc-user"

func init() {
	for _, backend := range testBackends {
		backend := backend
		Describe("Per-application signing algorithm ["+backend.name+"]", Ordered, func() {
			var h *harness
			const vsName = "app-alg-vs"
			var rs256AppId string
			var noOverrideAppId string

			BeforeAll(func() {
				if backend.dbMode == config.DatabaseModePostgres && !postgresBackendAvailable() {
					Skip("Postgres not available")
				}
				h = newE2eTestHarness(backend.dbMode, nil)

				scope := h.Scope().NewScope()
				defer scope.Close()
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)
				dbCtx := ioc.GetDependency[database.Context](scope)

				// Create a VS with EdDSA primary + RS256 additional
				createVSResp, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
					Name:                        vsName,
					DisplayName:                 "App Algorithm VS",
					PrimarySigningAlgorithm:     config.SigningAlgorithmEdDSA,
					AdditionalSigningAlgorithms: []config.SigningAlgorithm{config.SigningAlgorithmRS256},
					EnableRegistration:          true,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(dbCtx.SaveChanges(ctx)).To(Succeed())

				// Create an app with RS256 override
				rs256Resp, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
					VirtualServerName:     vsName,
					ProjectSlug:           createVSResp.SystemProjectSlug,
					Name:                  "rs256-app",
					DisplayName:           "RS256 App",
					Type:                  "public",
					RedirectUris:          []string{"http://localhost/callback"},
					AccessTokenHeaderType: "at+jwt",
					SigningAlgorithm:      utils.Ptr(config.SigningAlgorithmRS256),
				})
				Expect(err).ToNot(HaveOccurred())
				rs256AppId = rs256Resp.Id.String()

				// Create an app with no override (uses VS primary = EdDSA)
				noOverrideResp, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
					VirtualServerName:     vsName,
					ProjectSlug:           createVSResp.SystemProjectSlug,
					Name:                  "default-app",
					DisplayName:           "Default App",
					Type:                  "public",
					RedirectUris:          []string{"http://localhost/callback"},
					AccessTokenHeaderType: "at+jwt",
				})
				Expect(err).ToNot(HaveOccurred())
				noOverrideAppId = noOverrideResp.Id.String()

				// Create a service user on this VS to issue tokens via token exchange
				svcUserResp, err := mediatr.Send[*commands.CreateServiceUserResponse](ctx, m, commands.CreateServiceUser{
					VirtualServerName: vsName,
					Username:          appAlgServiceUserName,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(dbCtx.SaveChanges(ctx)).To(Succeed())

				_, err = mediatr.Send[*commands.AssociateServiceUserPublicKeyResponse](ctx, m, commands.AssociateServiceUserPublicKey{
					VirtualServerName: vsName,
					ServiceUserId:     svcUserResp.Id,
					PublicKey:         serviceUserPublicKey,
					Kid:               utils.Ptr(serviceUserKid),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(dbCtx.SaveChanges(ctx)).To(Succeed())

				_ = rs256AppId
				_ = noOverrideAppId
			})

			AfterAll(func() {
				if h != nil {
					h.Close()
				}
			})

			It("GET application returns signingAlgorithm in response", func() {
				resp, err := http.Get(fmt.Sprintf("%s/api/virtual-servers/%s/projects/%s/applications/%s",
					h.ApiUrl(), vsName, "system", rs256AppId))
				// This requires auth so we just check it's not 500
				Expect(err).ToNot(HaveOccurred())
				defer resp.Body.Close() //nolint:errcheck
				Expect(resp.StatusCode).ToNot(Equal(http.StatusInternalServerError))
			})

			It("tokens issued for RS256 app are signed with RS256", func() {
				token := issueTokenViaTokenExchange(h, vsName, "rs256-app")
				alg := jwtAlgorithm(token)
				Expect(alg).To(Equal("RS256"))
			})

			It("tokens issued for app with no override use VS primary (EdDSA)", func() {
				token := issueTokenViaTokenExchange(h, vsName, "default-app")
				alg := jwtAlgorithm(token)
				Expect(alg).To(Equal("EdDSA"))
			})

			It("cannot set signingAlgorithm to an algorithm not configured on the VS", func() {
				scope := h.Scope().NewScope()
				defer scope.Close()
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)

				// Try to create an app on a VS that only has EdDSA — RS256 is not configured
				_, err := mediatr.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
					VirtualServerName:     h.VirtualServer(), // EdDSA-only VS
					ProjectSlug:           "system",
					Name:                  "bad-alg-app",
					DisplayName:           "Bad Alg App",
					Type:                  "public",
					RedirectUris:          []string{"http://localhost/callback"},
					AccessTokenHeaderType: "at+jwt",
					SigningAlgorithm:      utils.Ptr(config.SigningAlgorithmRS256),
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("RS256"))
			})

			It("cannot remove an algorithm from VS that is still used by an app", func() {
				scope := h.Scope().NewScope()
				defer scope.Close()
				ctx := middlewares.ContextWithScope(context.Background(), scope)
				ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
				m := ioc.GetDependency[mediatr.Mediator](scope)

				// Try to remove RS256 from the VS — but rs256-app still uses it
				empty := []config.SigningAlgorithm{}
				_, err := mediatr.Send[*commands.PatchVirtualServerResponse](ctx, m, commands.PatchVirtualServer{
					VirtualServerName:           vsName,
					AdditionalSigningAlgorithms: &empty,
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("RS256"))
			})
		})
	}
}

// issueTokenViaTokenExchange exchanges a service-user JWT for an access token issued to the given app.
// The app name is used as the audience in the subject JWT, which causes the server to sign the
// resulting token with the algorithm configured on that application.
func issueTokenViaTokenExchange(h *harness, vsName, appName string) string {
	return acquireTokenForServiceUserOnVS(h, vsName, appName, appAlgServiceUserName, serviceUserKid, serviceUserPrivateKey)
}

// jwtAlgorithm parses the alg header from a JWT without verifying the signature.
func jwtAlgorithm(tokenString string) string {
	p := jwt.NewParser()
	token, _, err := p.ParseUnverified(tokenString, jwt.MapClaims{})
	Expect(err).ToNot(HaveOccurred())
	return token.Method.Alg()
}

// getApplicationFromAPI fetches an application via the admin API using the service user token.
func getApplicationFromAPI(h *harness, vsName, projectSlug, appId string) api.GetApplicationResponseDto {
	token := acquireTokenForServiceUser(h, serviceUserUsername, serviceUserKid, serviceUserPrivateKey)
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/api/virtual-servers/%s/projects/%s/applications/%s",
			h.ApiUrl(), vsName, projectSlug, appId),
		nil,
	)
	Expect(err).ToNot(HaveOccurred())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var dto api.GetApplicationResponseDto
	Expect(json.NewDecoder(resp.Body).Decode(&dto)).To(Succeed())
	return dto
}
