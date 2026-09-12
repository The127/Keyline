package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/jsonTypes"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/services"
	"github.com/The127/Keyline/utils"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/The127/ioc"
	"github.com/gorilla/mux"
)

const identityProviderLoginLifetime = 10 * time.Minute

type StartIdentityProviderLoginResponseDto struct {
	AuthorizationUrl string `json:"authorizationUrl"`
}

func identityProviderCallbackUrl(virtualServerName string, providerName string) string {
	return fmt.Sprintf("%s/oidc/%s/identity-providers/%s/callback", config.C.Server.ExternalUrl, virtualServerName, url.PathEscape(providerName))
}

// StartIdentityProviderLogin begins a login at an external identity provider for the login session.
// @Summary      Start identity provider login
// @Tags         Logins
// @Produce      json
// @Param        loginToken  path   string true  "Login session token"
// @Param        name        path   string true  "Identity provider name"
// @Success      200         {object}  handlers.StartIdentityProviderLoginResponseDto
// @Failure      401         {string}  string "Unknown token or wrong step"
// @Failure      404         {string}  string "Unknown identity provider"
// @Router       /logins/{loginToken}/identity-providers/{name}/start [post]
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

	identityProviderLogin, err := json.Marshal(jsonTypes.IdentityProviderLogin{
		LoginToken:   loginToken,
		ProviderName: identityProvider.Name(),
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(StartIdentityProviderLoginResponseDto{
		AuthorizationUrl: authorizationUrl,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
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
