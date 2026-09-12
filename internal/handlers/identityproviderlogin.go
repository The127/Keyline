package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/jsonTypes"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/services"
	"github.com/The127/Keyline/internal/services/identityproviders"
	"github.com/The127/Keyline/utils"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/The127/ioc"
	"github.com/gorilla/mux"
)

const identityProviderLoginLifetime = 10 * time.Minute

const identityProviderBrowserCookie = "keyline_idp_login"

func identityProviderCallbackUrl(virtualServerName string, providerName string) string {
	return fmt.Sprintf("%s/oidc/%s/identity-providers/%s/callback", config.C.Server.ExternalUrl, virtualServerName, url.PathEscape(providerName))
}

func identityProviderBrowserCookiePath(virtualServerName string) string {
	return fmt.Sprintf("/oidc/%s/identity-providers/", virtualServerName)
}

func newIdentityProviderBrowserCookie(virtualServerName string, browserToken string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     identityProviderBrowserCookie,
		Value:    browserToken,
		Path:     identityProviderBrowserCookiePath(virtualServerName),
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   strings.HasPrefix(config.C.Server.ExternalUrl, "https://"),
		SameSite: http.SameSiteLaxMode,
	}
}

// StartIdentityProviderLogin sends the browser to an external identity provider for the login session.
// @Summary      Start identity provider login
// @Tags         Logins
// @Param        loginToken  path   string true  "Login session token"
// @Param        name        path   string true  "Identity provider name"
// @Success      302         {string}  string "Redirect to the provider"
// @Failure      401         {string}  string "Unknown token or wrong step"
// @Failure      404         {string}  string "Unknown identity provider"
// @Router       /logins/{loginToken}/identity-providers/{name}/start [get]
func StartIdentityProviderLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]
	providerName := vars["name"]

	tokenService := ioc.GetDependency[services.TokenService](scope)
	rawLoginInfo, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	switch {
	case errors.Is(err, services.ErrTokenNotFound):
		http.Error(w, "unknown token", http.StatusUnauthorized)
		return

	case err != nil:
		utils.HandleHttpError(w, fmt.Errorf("getting token: %w", err))
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(rawLoginInfo), &loginInfo)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	if loginInfo.Step != jsonTypes.LoginStepPasswordVerification {
		http.Error(w, "login already identified a user", http.StatusUnauthorized)
		return
	}

	dbContext := ioc.GetDependency[database.Context](scope)
	identityProviderFilter := repositories.NewIdentityProviderFilter().
		VirtualServerId(loginInfo.VirtualServerId).
		Name(providerName)
	identityProvider, err := dbContext.IdentityProviders().FirstOrErr(ctx, identityProviderFilter)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	state := base64.RawURLEncoding.EncodeToString(utils.GetSecureRandomBytes(32))
	codeVerifier := base64.RawURLEncoding.EncodeToString(utils.GetSecureRandomBytes(32))
	nonce := base64.RawURLEncoding.EncodeToString(utils.GetSecureRandomBytes(16))
	browserToken := base64.RawURLEncoding.EncodeToString(utils.GetSecureRandomBytes(32))

	identityProviderLogin, err := json.Marshal(jsonTypes.IdentityProviderLogin{
		LoginToken:   loginToken,
		ProviderName: identityProvider.Name(),
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		BrowserToken: browserToken,
	})
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("marshaling identity provider login: %w", err))
		return
	}

	if loginInfo.IdentityProviderState != "" {
		err = tokenService.DeleteToken(ctx, services.IdentityProviderLoginTokenType, loginInfo.IdentityProviderState)
		if err != nil && !errors.Is(err, services.ErrTokenNotFound) {
			utils.HandleHttpError(w, fmt.Errorf("forgetting previous identity provider login: %w", err))
			return
		}
	}

	err = tokenService.StoreToken(ctx, services.IdentityProviderLoginTokenType, state, string(identityProviderLogin), identityProviderLoginLifetime)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("storing identity provider login: %w", err))
		return
	}

	loginInfo.IdentityProviderState = state
	updatedLoginInfo, err := json.Marshal(loginInfo)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("marshaling login info: %w", err))
		return
	}
	err = tokenService.UpdateToken(ctx, services.LoginSessionTokenType, loginToken, string(updatedLoginInfo), 15*time.Minute)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("updating login info: %w", err))
		return
	}

	authorizationUrl, err := buildAuthorizationUrl(identityProvider, loginInfo.VirtualServerName, state, codeVerifier, nonce)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	http.SetCookie(w, newIdentityProviderBrowserCookie(loginInfo.VirtualServerName, browserToken, int(identityProviderLoginLifetime.Seconds())))
	http.Redirect(w, r, authorizationUrl, http.StatusFound)
}

func buildAuthorizationUrl(identityProvider *repositories.IdentityProvider, virtualServerName string, state string, codeVerifier string, nonce string) (string, error) {
	settings := identityProvider.Settings()

	authorizationUrl, err := url.Parse(settings.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("parsing authorization endpoint of %s: %w", identityProvider.Name(), err)
	}

	codeChallenge := sha256.Sum256([]byte(codeVerifier))

	query := authorizationUrl.Query()
	query.Set("client_id", settings.ClientId)
	query.Set("redirect_uri", identityProviderCallbackUrl(virtualServerName, identityProvider.Name()))
	query.Set("response_type", "code")
	query.Set("scope", strings.Join(settings.Scopes, " "))
	query.Set("state", state)
	query.Set("code_challenge", base64.RawURLEncoding.EncodeToString(codeChallenge[:]))
	query.Set("code_challenge_method", "S256")
	query.Set("nonce", nonce)
	authorizationUrl.RawQuery = query.Encode()

	return authorizationUrl.String(), nil
}

const identityProviderErrorCode = "identity_provider"

func loginPageUrl(loginToken string, errorCode string) string {
	query := url.Values{}
	query.Set("token", loginToken)
	if errorCode != "" {
		query.Set("error", errorCode)
	}
	return fmt.Sprintf("%s/login?%s", config.C.Frontend.ExternalUrl, query.Encode())
}

// IdentityProviderCallback finishes a login at an external identity provider and returns the browser to the login page.
// @Summary      Identity provider callback
// @Tags         OIDC
// @Param        virtualServerName  path   string true  "Virtual server name"  default(keyline)
// @Param        name               path   string true  "Identity provider name"
// @Param        code               query  string false "Authorization code"
// @Param        state              query  string true  "State from the start"
// @Success      302                {string}  string "Redirect to the login page"
// @Failure      400                {string}  string "Unknown state"
// @Router       /oidc/{virtualServerName}/identity-providers/{name}/callback [get]
func IdentityProviderCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	virtualServerName := vars["virtualServerName"]
	providerName := vars["name"]
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	tokenService := ioc.GetDependency[services.TokenService](scope)
	rawIdentityProviderLogin, err := tokenService.GetToken(ctx, services.IdentityProviderLoginTokenType, state)
	switch {
	case errors.Is(err, services.ErrTokenNotFound):
		http.Error(w, "unknown state", http.StatusBadRequest)
		return

	case err != nil:
		utils.HandleHttpError(w, fmt.Errorf("getting identity provider login: %w", err))
		return
	}

	err = tokenService.DeleteToken(ctx, services.IdentityProviderLoginTokenType, state)
	switch {
	case errors.Is(err, services.ErrTokenNotFound):
		http.Error(w, "unknown state", http.StatusBadRequest)
		return

	case err != nil:
		utils.HandleHttpError(w, fmt.Errorf("consuming identity provider login: %w", err))
		return
	}

	var identityProviderLogin jsonTypes.IdentityProviderLogin
	err = json.Unmarshal([]byte(rawIdentityProviderLogin), &identityProviderLogin)
	if err != nil {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	browserCookie, err := r.Cookie(identityProviderBrowserCookie)
	if err != nil || browserCookie.Value == "" || browserCookie.Value != identityProviderLogin.BrowserToken {
		http.Error(w, "callback from another browser", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, newIdentityProviderBrowserCookie(virtualServerName, "", -1))

	loginToken := identityProviderLogin.LoginToken
	rawLoginInfo, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		http.Error(w, "login expired", http.StatusBadRequest)
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(rawLoginInfo), &loginInfo)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	failLogin := func(reason error) {
		logging.Logger.Warnf("identity provider login failed: %v", reason)
		http.Redirect(w, r, loginPageUrl(loginToken, identityProviderErrorCode), http.StatusFound)
	}

	if identityProviderLogin.ProviderName != providerName {
		failLogin(fmt.Errorf("callback for %s arrived at %s", identityProviderLogin.ProviderName, providerName))
		return
	}
	if loginInfo.VirtualServerName != virtualServerName {
		failLogin(fmt.Errorf("callback for %s arrived at %s", loginInfo.VirtualServerName, virtualServerName))
		return
	}
	if loginInfo.IdentityProviderState != state {
		failLogin(fmt.Errorf("state is not the current one of the login"))
		return
	}
	if loginInfo.Step != jsonTypes.LoginStepPasswordVerification {
		failLogin(fmt.Errorf("login already identified a user"))
		return
	}
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		failLogin(fmt.Errorf("provider answered %s: %s", providerError, r.URL.Query().Get("error_description")))
		return
	}
	if code == "" {
		failLogin(fmt.Errorf("callback without a code"))
		return
	}

	dbContext := ioc.GetDependency[database.Context](scope)
	identityProviderFilter := repositories.NewIdentityProviderFilter().
		VirtualServerId(loginInfo.VirtualServerId).
		Name(providerName)
	identityProvider, err := dbContext.IdentityProviders().FirstOrNil(ctx, identityProviderFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting identity provider: %w", err))
		return
	}
	if identityProvider == nil {
		failLogin(fmt.Errorf("identity provider %s no longer exists", providerName))
		return
	}

	upstream := ioc.GetDependency[*identityproviders.Client](scope)
	settings := identityProvider.Settings()

	tokens, err := upstream.Exchange(ctx, settings, code, identityProviderCallbackUrl(virtualServerName, providerName), identityProviderLogin.CodeVerifier)
	if err != nil {
		failLogin(err)
		return
	}

	identity, err := resolveExternalIdentity(ctx, upstream, settings, tokens, identityProviderLogin.Nonce)
	if err != nil {
		failLogin(err)
		return
	}

	rawLoginInfo, err = tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		http.Error(w, "login expired", http.StatusBadRequest)
		return
	}
	err = json.Unmarshal([]byte(rawLoginInfo), &loginInfo)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}
	if loginInfo.IdentityProviderState != state || loginInfo.Step != jsonTypes.LoginStepPasswordVerification {
		failLogin(fmt.Errorf("login changed while the provider was answering"))
		return
	}

	credentialFilter := repositories.NewCredentialFilter().
		Type(repositories.CredentialTypeExternalIdentity).
		DetailIdentityProviderId(identityProvider.Id()).
		DetailSubject(identity.Subject)
	credential, err := dbContext.Credentials().FirstOrNil(ctx, credentialFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting external identity: %w", err))
		return
	}
	if credential == nil {
		failLogin(fmt.Errorf("no user is linked to subject %s at %s", identity.Subject, providerName))
		return
	}

	userFilter := repositories.NewUserFilter().
		VirtualServerId(loginInfo.VirtualServerId).
		Id(credential.UserId())
	user, err := dbContext.Users().FirstOrNil(ctx, userFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting user: %w", err))
		return
	}
	if user == nil {
		failLogin(fmt.Errorf("linked user %s is not in virtual server %s", credential.UserId(), virtualServerName))
		return
	}

	loginInfo.UserId = user.Id()
	loginInfo.IdentityProviderState = ""
	loginInfo.Step, err = DetermineNextLoginStep(ctx, &loginInfo)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("determining next login step: %w", err))
		return
	}

	updatedLoginInfo, err := json.Marshal(loginInfo)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("marshaling login info: %w", err))
		return
	}
	err = tokenService.UpdateToken(ctx, services.LoginSessionTokenType, loginToken, string(updatedLoginInfo), 15*time.Minute)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("updating login info: %w", err))
		return
	}

	http.Redirect(w, r, loginPageUrl(loginToken, ""), http.StatusFound)
}

type externalIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Username      string
}

func resolveExternalIdentity(ctx context.Context, upstream *identityproviders.Client, settings repositories.IdentityProviderSettings, tokens identityproviders.Tokens, nonce string) (externalIdentity, error) {
	claims := map[string]any{}

	if settings.Issuer != "" {
		if tokens.IdToken == "" {
			return externalIdentity{}, fmt.Errorf("provider with issuer %s returned no id token", settings.Issuer)
		}
		idTokenClaims, err := upstream.VerifyIdToken(ctx, settings, tokens.IdToken, nonce)
		if err != nil {
			return externalIdentity{}, err
		}
		for name, value := range idTokenClaims {
			claims[name] = value
		}
	}

	userinfo, err := upstream.Userinfo(ctx, settings, tokens.AccessToken)
	if err != nil {
		return externalIdentity{}, err
	}
	if idTokenSubject, ok := claims["sub"]; ok {
		if userinfoSubject, ok := userinfo["sub"]; ok && userinfoSubject != idTokenSubject {
			return externalIdentity{}, fmt.Errorf("userinfo subject differs from the id token subject")
		}
	}
	for name, value := range userinfo {
		claims[name] = value
	}

	subject, _ := claims["sub"].(string)
	if subject == "" {
		return externalIdentity{}, fmt.Errorf("provider returned no subject")
	}

	email, _ := claims["email"].(string)
	emailVerified, _ := claims["email_verified"].(bool)
	name, _ := claims["name"].(string)
	username, _ := claims["preferred_username"].(string)

	return externalIdentity{
		Subject:       subject,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		Username:      username,
	}, nil
}
