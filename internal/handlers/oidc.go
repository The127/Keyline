package handlers

import (
	"Keyline/internal/clock"
	"Keyline/internal/config"
	"Keyline/internal/jsonTypes"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"Keyline/internal/services/claimsMapping"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type OidcError struct {
	Error            string
	ErrorDescription string
	ErrorUri         string
}

var (
	loginRequired = OidcError{
		Error:            "login_required",
		ErrorDescription: "The Authorization Server requires End-User authentication",
		ErrorUri:         "https://openid.net/specs/openid-connect-core-1_0.html#AuthError",
	}
	unsupportedResponseType = OidcError{
		Error:            "unsupported_response_type",
		ErrorDescription: "The authorization server does not support obtaining an authorization code using this method.",
		ErrorUri:         "https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2.1",
	}
)

type Ed25519JWK struct {
	Kty string `json:"kty"` // Key Type
	Crv string `json:"crv"` // Curve
	Alg string `json:"alg"` // Algorithm
	Use string `json:"use"` // Use (sig = signature)
	Kid string `json:"kid"` // Key ID
	X   string `json:"x"`   // Public key (base64url)
}

type RS256JWK struct {
	Kty string `json:"kty"` // Key Type, e.g. "RSA"
	Alg string `json:"alg"` // Algorithm, e.g. "RS256"
	Use string `json:"use"` // Public key use, usually "sig"
	Kid string `json:"kid"` // Key ID
	N   string `json:"n"`   // Modulus, base64url encoded
	E   string `json:"e"`   // Exponent, base64url encoded
}

type JwksResponseDto struct {
	Keys []any `json:"keys"`
}

func trimLeadingZeros(b []byte) []byte {
	for i := 0; i < len(b); i++ {
		if b[i] != 0 {
			return b[i:]
		}
	}
	return []byte{0}
}

// WellKnownJwks returns the JSON Web Key Set (JWKS) for a virtual server.
// @Summary      JWKS for virtual server
// @Description  Returns the public keys used to verify tokens for this virtual server.
// @Tags         OIDC
// @Produce      json
// @Param        virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @Success      200  {object}  handlers.JwksResponseDto
// @Failure      400  {string}  string
// @Failure      500  {string}  string
// @Router       /oidc/{virtualServerName}/.well-known/jwks.json [get]
func WellKnownJwks(w http.ResponseWriter, r *http.Request) {
	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(r.Context())
	keyService := ioc.GetDependency[services.KeyService](scope)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(vsName)
	virtualServer, err := virtualServerRepository.First(r.Context(), virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}

	keyPair := keyService.GetKey(vsName, virtualServer.SigningAlgorithm())

	kid := keyPair.GetKid()

	keys := make([]any, 0)

	switch virtualServer.SigningAlgorithm() {
	case config.SigningAlgorithmEdDSA:
		keys = append(keys, Ed25519JWK{
			Kty: "OKP",
			Crv: "Ed25519",
			Alg: "EdDSA",
			Use: "sig",
			Kid: kid,
			X:   base64.RawURLEncoding.EncodeToString(keyPair.PublicKeyBytes()),
		})

	case config.SigningAlgorithmRS256:
		rsaPublicKey := keyPair.PublicKey().(*rsa.PublicKey)

		eBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(eBytes, uint64(rsaPublicKey.E))

		// trim leading zero bytes (JWK requires minimal representation)
		eBytes = trimLeadingZeros(eBytes)

		keys = append(keys, RS256JWK{
			Kty: "RSA",
			Alg: "RS256",
			Use: "sig",
			Kid: kid,
			N:   base64.RawURLEncoding.EncodeToString(rsaPublicKey.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(eBytes),
		})

	default:
		utils.HandleHttpError(w, fmt.Errorf("unsupported signing algorithm: %s", virtualServer.SigningAlgorithm()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(JwksResponseDto{
		Keys: keys,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type OpenIdConfigurationResponseDto struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	EndSessionEndpoint                string   `json:"end_session_endpoint"`
	JwksUri                           string   `json:"jwks_uri"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IdTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	RequestParameterSupported         bool     `json:"request_parameter_supported"`
}

// WellKnownOpenIdConfiguration exposes the OIDC discovery document.
// @Summary      OpenID Provider configuration
// @Tags         OIDC
// @Produce      json
// @Param        virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @Success      200  {object}  handlers.OpenIdConfigurationResponseDto
// @Failure      400  {string}  string
// @Router       /oidc/{virtualServerName}/.well-known/openid-configuration [get]
func WellKnownOpenIdConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	// Fetch the virtual server from database
	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(vsName)
	virtualServer, err := virtualServerRepository.First(ctx, virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}
	if virtualServer == nil {
		utils.HandleHttpError(w, fmt.Errorf("virtual server not found"))
		return
	}

	responseDto := OpenIdConfigurationResponseDto{
		Issuer: fmt.Sprintf("%s/oidc/%s", config.C.Server.ExternalUrl, vsName),

		AuthorizationEndpoint: fmt.Sprintf("%s/oidc/%s/authorize", config.C.Server.ExternalUrl, vsName),
		TokenEndpoint:         fmt.Sprintf("%s/oidc/%s/token", config.C.Server.ExternalUrl, vsName),
		UserinfoEndpoint:      fmt.Sprintf("%s/oidc/%s/userinfo", config.C.Server.ExternalUrl, vsName),
		EndSessionEndpoint:    fmt.Sprintf("%s/oidc/%s/end_session", config.C.Server.ExternalUrl, vsName),
		JwksUri:               fmt.Sprintf("%s/oidc/%s/.well-known/jwks.json", config.C.Server.ExternalUrl, vsName),

		ResponseTypesSupported:            []string{"code"}, // TODO: maybe support more
		RequestParameterSupported:         true,
		SubjectTypesSupported:             []string{"public"},
		IdTokenSigningAlgValuesSupported:  []string{string(virtualServer.SigningAlgorithm())},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post"},

		ScopesSupported: []string{"openid", "email", "profile"}, // TODO: get from db
		ClaimsSupported: []string{"sub", "name", "email"},       // TODO: get from db
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(responseDto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type AuthorizationRequest struct {
	ResponseTypes       []string
	VirtualServerName   string
	ApplicationName     string
	RedirectUri         string
	Scopes              []string
	State               string
	Nonce               string
	ResponseMode        string
	PKCEChallenge       string
	PKCEChallengeMethod string
}

// BeginAuthorizationFlow starts the OIDC authorization code flow.
// @Summary      Authorize
// @Description  Starts the Authorization Code flow. If the user is not authenticated, redirects to your login UI; otherwise redirects to the application's redirect_uri with an authorization code.
// @Tags         OIDC
// @Produce      plain
// @Accept       application/x-www-form-urlencoded
// @Param        virtualServerName      path     string true   "Virtual server name"  default(keyline)
// @Param        response_type          query    string true   "Must be 'code'"
// @Param        client_id              query    string true   "Application (client) ID"
// @Param        redirect_uri           query    string true   "Registered redirect URI"
// @Param        scope                  query    string true   "Space-delimited scopes (must include 'openid')"
// @Param        state                  query    string false  "Opaque value returned to client"
// @Param        response_mode          query    string false  "e.g. 'query'"
// @Param        code_challenge         query    string false  "PKCE code challenge"
// @Param        code_challenge_method  query    string false  "S256 or plain" Enums(S256,plain)
// @Success      302  {string}  string  "Redirect to redirect_uri with code (& state)"
// @Failure      400  {string}  string
// @Router       /oidc/{virtualServerName}/authorize [get]
// @Router       /oidc/{virtualServerName}/authorize [post]
func BeginAuthorizationFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	err := r.ParseForm()
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	prompt := r.Form.Get("prompt")
	authRequest := AuthorizationRequest{
		ResponseTypes:       strings.Split(r.Form.Get("response_type"), " "),
		VirtualServerName:   vsName,
		ApplicationName:     r.Form.Get("client_id"),
		RedirectUri:         r.Form.Get("redirect_uri"),
		Scopes:              strings.Split(r.Form.Get("scope"), " "),
		State:               r.Form.Get("state"),
		Nonce:               r.Form.Get("nonce"),
		ResponseMode:        r.Form.Get("response_mode"),
		PKCEChallenge:       r.Form.Get("code_challenge"),
		PKCEChallengeMethod: r.Form.Get("code_challenge_method"),
	}

	requestParam := r.Form.Get("request")
	if requestParam != "" {
		token, _, err := new(jwt.Parser).ParseUnverified(requestParam, jwt.MapClaims{})
		if err != nil {
			utils.HandleHttpError(w, fmt.Errorf("parsing request parameter: %w", err))
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if claims["response_type"] != nil {
				authRequest.ResponseTypes = strings.Split(claims["response_type"].(string), " ")
			}
			if claims["client_id"] != nil {
				authRequest.ApplicationName = claims["client_id"].(string)
			}
			if claims["redirect_uri"] != nil {
				authRequest.RedirectUri = claims["redirect_uri"].(string)
			}
			if claims["scope"] != nil {
				authRequest.Scopes = strings.Split(claims["scope"].(string), " ")
			}
			if claims["state"] != nil {
				authRequest.State = claims["state"].(string)
			}
			if claims["nonce"] != nil {
				authRequest.Nonce = claims["nonce"].(string)
			}
			if claims["response_mode"] != nil {
				authRequest.ResponseMode = claims["response_mode"].(string)
			}
		}

	}

	// TODO: use validation annotations to validate the auth request

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(vsName)
	virtualServer, err := virtualServerRepository.First(ctx, virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}

	if virtualServer == nil {
		utils.HandleHttpError(w, fmt.Errorf("virtual server not found"))
		return
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		Name(authRequest.ApplicationName).
		VirtualServerId(virtualServer.Id())
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting application: %w", err))
		return
	}

	if application == nil {
		utils.HandleHttpError(w, fmt.Errorf("application not found"))
		return
	}

	if application.RedirectUris() == nil || len(application.RedirectUris()) == 0 {
		utils.HandleHttpError(w, fmt.Errorf("application has no redirect uris"))
		return
	}

	redirectOk := false
	for _, allowed := range application.RedirectUris() {
		if authRequest.RedirectUri == allowed {
			redirectOk = true
			break
		}
	}
	if !redirectOk {
		utils.HandleHttpError(w, fmt.Errorf("redirect uri does not match registered uris"))
		return
	}

	if len(authRequest.ResponseTypes) != 1 || authRequest.ResponseTypes[0] != "code" {
		errorRedirect(w, r, authRequest, unsupportedResponseType)
		return
	}

	if !slices.Contains(authRequest.Scopes, "openid") {
		utils.HandleHttpError(w, fmt.Errorf("required openid scope missing"))
		return
	}

	// TODO: check the scopes for email and profile

	tokenService := ioc.GetDependency[services.TokenService](scope)

	s, ok := middlewares.GetSession(ctx)
	if ok {
		// TODO: consent page

		codeInfo := jsonTypes.NewCodeInfo(virtualServer.Name(), authRequest.Scopes, s.UserId(), authRequest.Nonce, s.CreatedAt())

		codeInfoString, err := json.Marshal(codeInfo)
		if err != nil {
			utils.HandleHttpError(w, fmt.Errorf("marshaling code info: %w", err))
			return
		}

		code, err := tokenService.GenerateAndStoreToken(ctx, services.OidcCodeTokenType, string(codeInfoString), time.Minute)
		if err != nil {
			utils.HandleHttpError(w, fmt.Errorf("generating code: %w", err))
			return
		}

		redirectUri, err := url.Parse(authRequest.RedirectUri)
		if err != nil {
			utils.HandleHttpError(w, fmt.Errorf("parsing redirect uri: %w", err))
			return
		}

		query := redirectUri.Query()
		query.Set("code", code)

		if authRequest.State != "" {
			query.Set("state", authRequest.State)
		}

		redirectUri.RawQuery = query.Encode()

		// redirect to that uri with code
		http.Redirect(w, r, redirectUri.String(), http.StatusFound)
		return
	}

	if prompt == "none" {
		errorRedirect(w, r, authRequest, loginRequired)
		return
	}

	loginInfo := jsonTypes.NewLoginInfo(
		virtualServer,
		application,
		r.URL.String(),
	)

	loginInfoString, err := json.Marshal(loginInfo)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("marshaling login info: %w", err))
		return
	}

	loginSessionToken, err := tokenService.GenerateAndStoreToken(ctx, services.LoginSessionTokenType, string(loginInfoString), time.Minute*15)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("generating login session token: %w", err))
		return
	}

	redirectUrl := fmt.Sprintf(
		"%s/login?token=%s",
		config.C.Frontend.ExternalUrl,
		loginSessionToken,
	)
	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

func errorRedirect(w http.ResponseWriter, r *http.Request, authRequest AuthorizationRequest, oidcError OidcError) {
	errorUrl, err := url.Parse(authRequest.RedirectUri)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("parsing redirect uri: %w", err))
		return
	}

	query := errorUrl.Query()
	query.Set("error", oidcError.Error)

	if oidcError.ErrorDescription != "" {
		query.Set("error_description", oidcError.ErrorDescription)
	}

	if oidcError.ErrorUri != "" {
		query.Set("error_uri", oidcError.ErrorUri)
	}

	if authRequest.State != "" {
		query.Set("state", authRequest.State)
	}

	errorUrl.RawQuery = query.Encode()

	http.Redirect(w, r, errorUrl.String(), http.StatusFound)
}

// OidcEndSession ends the user session and redirects.
// @Summary      End session
// @Tags         OIDC
// @Produce      json
// @Param        virtualServerName         path     string true  "Virtual server name"  default(keyline)
// @Param        id_token_hint             query    string true  "ID token hint of the current session"
// @Param        post_logout_redirect_uri  query    string false "Where to redirect after logout (must be registered)"
// @Param        state                     query    string false "Opaque value returned to client"
// @Success      302  {string}  string "Redirect to post_logout_redirect_uri"
// @Failure      400  {string}  string
// @Router       /oidc/{virtualServerName}/end_session [get]
func OidcEndSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	err := r.ParseForm()
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	state := r.Form.Get("state")

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(vsName)
	virtualServer, err := virtualServerRepository.First(r.Context(), virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(vsName, virtualServer.SigningAlgorithm())

	idTokenString := r.Form.Get("id_token_hint")
	if idTokenString == "" {
		utils.HandleHttpError(w, fmt.Errorf("id token hint not found"))
		return
	}

	idToken, err := jwt.Parse(idTokenString, func(token *jwt.Token) (interface{}, error) {
		jwtSigningMethod, err := getJwtSigningMethod(virtualServer.SigningAlgorithm())
		if err != nil {
			return nil, fmt.Errorf("getting jwt signing method: %w", err)
		}

		tokenMethodAlgorithm := token.Method.Alg()
		if jwtSigningMethod.Alg() != tokenMethodAlgorithm {
			return nil, fmt.Errorf("unexpected signing method: %v", tokenMethodAlgorithm)
		}

		return keyPair.PublicKey(), nil
	})
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("parsing id token: %w", err))
		return
	}

	if !idToken.Valid {
		utils.HandleHttpError(w, fmt.Errorf("id token is not valid"))
		return
	}

	idTokenClaims := idToken.Claims.(jwt.MapClaims)
	clientId, err := extractClientIdFromJwt(idTokenClaims)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("extracting client id from id token: %w", err))
		return
	}

	if clientId == "" {
		utils.HandleHttpError(w, fmt.Errorf("client id not found"))
		return
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().Name(clientId)
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting application: %w", err))
		return
	}
	if application == nil {
		utils.HandleHttpError(w, fmt.Errorf("application not found"))
		return
	}

	redirectUriString := r.Form.Get("post_logout_redirect_uri")
	if redirectUriString == "" {
		redirectUriString = application.RedirectUris()[0]
	}

	if !slices.Contains(application.PostLogoutRedirectUris(), redirectUriString) {
		utils.HandleHttpError(w, fmt.Errorf("redirect uri not found"))
		return
	}

	err = middlewares.DeleteSession(w, r, vsName)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	redirectUri, err := url.Parse(redirectUriString)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("parsing redirect uri: %w", err))
		return
	}

	query := redirectUri.Query()

	if state != "" {
		query.Set("state", state)
	}

	redirectUri.RawQuery = query.Encode()

	http.Redirect(w, r, redirectUriString, http.StatusFound)
}

func extractClientIdFromJwt(idTokenClaims jwt.MapClaims) (string, error) {
	clientIdString, ok := idTokenClaims["aud"]
	if !ok {
		return "", fmt.Errorf("client id not found")
	}

	clientId, ok := clientIdString.(string)
	if !ok {
		clientIds, ok := clientIdString.([]interface{})
		if !ok {
			return "", fmt.Errorf("expected string or array of strings for client id")
		}

		if len(clientIds) != 1 {
			return "", fmt.Errorf("expected array of length 1 for client id")
		}

		clientId, ok = clientIds[0].(string)
		if !ok {
			return "", fmt.Errorf("expected string for client id")
		}
	}

	return clientId, nil
}

type OidcUserInfoResponseDto struct {
	Sub           string `json:"sub"`
	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
}

// OidcUserinfo returns the userinfo for the presented access token.
// @Summary      Userinfo
// @Tags         OIDC
// @Produce      json
// @Param        virtualServerName  path   string true  "Virtual server name"  default(keyline)
// @Security     BearerAuth
// @Success      200  {object}  handlers.OidcUserInfoResponseDto
// @Failure      401  {string}  string
// @Router       /oidc/{virtualServerName}/userinfo [get, post]
func OidcUserinfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(vsName)
	virtualServer, err := virtualServerRepository.First(r.Context(), virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(vsName, virtualServer.SigningAlgorithm())

	err = r.ParseForm()
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	bearer, err := extractAccessToken(r)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("extracting access token: %w", err))
		return
	}
	if bearer == "" {
		utils.HandleHttpError(w, fmt.Errorf("authorization header not found"))
		return
	}

	tokenString := strings.TrimPrefix(bearer, "Bearer ")
	if tokenString == "" {
		utils.HandleHttpError(w, fmt.Errorf("token not found"))
		return
	}

	tokenJwt, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return keyPair.PublicKey(), nil
	})
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("parsing token: %w", err))
		return
	}

	subject, err := tokenJwt.Claims.GetSubject()
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting subject: %w", err))
		return
	}

	userId, err := uuid.Parse(subject)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("parsing subject: %w", err))
		return
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Id(userId).VirtualServerId(virtualServer.Id())
	user, err := userRepository.First(ctx, userFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting user: %w", err))
		return
	}
	if user == nil {
		utils.HandleHttpError(w, fmt.Errorf("user not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	tempResult := OidcUserInfoResponseDto{
		Sub: userId.String(),
	}

	scopes, err := extractScopes(tokenJwt)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("extracting scopes: %w", err))
		return
	}

	if slices.Contains(scopes, "email") {
		tempResult.Email = user.PrimaryEmail()
		tempResult.EmailVerified = utils.Ptr(user.EmailVerified())
	}

	if slices.Contains(scopes, "profile") {
		tempResult.Name = user.DisplayName()
	}

	err = json.NewEncoder(w).Encode(tempResult)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

func extractScopes(tokenJwt *jwt.Token) ([]string, error) {
	claims := tokenJwt.Claims.(jwt.MapClaims)
	scopesClaim, ok := claims["scopes"]
	if !ok {
		return []string{}, nil
	}

	scopes, ok := scopesClaim.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected array of strings for scope")
	}

	scopeStrings := make([]string, 0, len(scopes))
	for _, t := range scopes {
		s, ok := t.(string)
		if !ok {
			return nil, fmt.Errorf("scope value is not a string: %v", t)
		}
		scopeStrings = append(scopeStrings, s)
	}

	return scopeStrings, nil
}

func extractAccessToken(r *http.Request) (string, error) {
	bearer := r.Header.Get("Authorization")
	if bearer != "" {
		if !strings.HasPrefix(bearer, "Bearer ") {
			return "", fmt.Errorf("authorization header is not a bearer token")
		}

		return bearer, nil
	}

	return r.PostFormValue("access_token"), nil
}

// OidcToken exchanges authorization code or refresh token for tokens.
// @Summary      Token endpoint
// @Tags         OIDC
// @Accept       application/x-www-form-urlencoded
// @Produce      json
// @Param        grant_type    formData  string true  "authorization_code | refresh_token"
// @Param        code          formData  string false "Required when grant_type=authorization_code"
// @Param        refresh_token formData  string false "Required when grant_type=refresh_token"
// @Param        client_id     formData  string false "If no Authorization header"
// @Security     BasicAuth
// @Success      200  {object}  handlers.CodeFlowResponse      "When grant_type=authorization_code"
// @Success      200  {object}  handlers.RefreshTokenResponse  "When grant_type=refresh_token"
// @Success      200  {object}  handlers.TokenExchangeResponse "When grant_type=urn:ietf:params:oauth:grant-type:token-exchange"
// @Failure      400  {string}  string
// @Router       /oidc/{virtualServerName}/token [post]
func OidcToken(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	grantType := r.Form.Get("grant_type")

	switch grantType {
	case "authorization_code":
		handleAuthorizationCode(w, r)

	case "refresh_token":
		handleRefreshToken(w, r)

	case "urn:ietf:params:oauth:grant-type:token-exchange":
		handleTokenExchange(w, r)

	default:
		utils.HandleHttpError(w, fmt.Errorf("unsupported grant type: %s", grantType))
		return
	}
}

//nolint:unparam
func authenticateApplication(ctx context.Context, applicationName string, applicationSecret string) (*repositories.Application, error) {
	scope := middlewares.GetScope(ctx)

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().Name(applicationName)
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}
	if application == nil {
		return nil, fmt.Errorf("application not found")
	}

	if applicationSecret == "" {
		// TODO: do pkce
		return application, nil
	}

	hashedSecret := application.HashedSecret()
	if utils.CheapCompareHash(applicationSecret, hashedSecret) {
		return application, nil
	}

	return nil, fmt.Errorf("invalid secret")
}

type CodeFlowResponse struct {
	TokenType    string `json:"token_type"`
	IdToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}

func writeOAuthError(w http.ResponseWriter, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

func handleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	code := r.Form.Get("code")

	tokenService := ioc.GetDependency[services.TokenService](scope)
	valueString, err := tokenService.GetToken(ctx, services.OidcCodeTokenType, code)
	if err != nil {
		writeOAuthError(w, "invalid_grant", "authorization code already used")
		return
	}

	var codeInfo jsonTypes.CodeInfo
	err = json.Unmarshal([]byte(valueString), &codeInfo)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("unmarshaling code info: %w", err))
		return
	}

	clientId, clientSecret, hasBasicAuth := r.BasicAuth()
	if !hasBasicAuth {
		clientId = r.Form.Get("client_id")
	}

	_, err = authenticateApplication(ctx, clientId, clientSecret)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("authenticating application: %w", err))
		return
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Id(codeInfo.UserId)
	user, err := userRepository.First(ctx, userFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting user: %w", err))
		return
	}
	if user == nil {
		utils.HandleHttpError(w, fmt.Errorf("user not found"))
		return
	}

	// TODO: get claims from scopes

	clockService := ioc.GetDependency[clock.Service](scope)
	now := clockService.Now()

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(codeInfo.VirtualServerName)
	virtualServer, err := virtualServerRepository.First(r.Context(), virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		Name(clientId)
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting application: %w", err))
		return
	}
	if application == nil {
		utils.HandleHttpError(w, fmt.Errorf("application not found"))
		return
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(codeInfo.VirtualServerName, virtualServer.SigningAlgorithm())

	tokenDuration := time.Hour // TODO: make this configurable per virtual server

	params := TokenGenerationParams{
		UserId:             codeInfo.UserId,
		VirtualServerName:  codeInfo.VirtualServerName,
		ClientId:           clientId,
		ApplicationId:      application.Id(),
		GrantedScopes:      codeInfo.GrantedScopes,
		UserDisplayName:    user.DisplayName(),
		UserPrimaryEmail:   user.PrimaryEmail(),
		ExternalUrl:        config.C.Server.ExternalUrl,
		KeyPair:            keyPair,
		IssuedAt:           now,
		AccessTokenExpiry:  tokenDuration,
		IdTokenExpiry:      tokenDuration,
		RefreshTokenExpiry: tokenDuration,
		Nonce:              codeInfo.Nonce,
		AuthenticatedAt:    codeInfo.AuthenticatedAt,
	}

	tokens, err := generateTokens(ctx, params, tokenService)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = tokenService.DeleteToken(ctx, services.OidcCodeTokenType, code)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("deleting code: %w", err))
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	scopeString := strings.Join(codeInfo.GrantedScopes, " ")
	response := CodeFlowResponse{
		TokenType:    "Bearer",
		IdToken:      tokens.IdToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Scope:        scopeString,
		ExpiresIn:    tokens.ExpiresIn,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("encoding response: %w", err))
		return
	}
}

func getJwtSigningMethod(algorithm config.SigningAlgorithm) (jwt.SigningMethod, error) {
	switch algorithm {
	case config.SigningAlgorithmEdDSA:
		return jwt.SigningMethodEdDSA, nil

	case config.SigningAlgorithmRS256:
		return jwt.SigningMethodRS256, nil

	default:
		return nil, fmt.Errorf("unsupported signing algorithm: %s", algorithm)
	}
}

type RefreshTokenResponse struct {
	TokenType    string `json:"token_type"`
	IdToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type TokenGenerationParams struct {
	UserId             uuid.UUID
	VirtualServerName  string
	ClientId           string
	ApplicationId      uuid.UUID
	GrantedScopes      []string
	UserDisplayName    string
	UserPrimaryEmail   string
	ExternalUrl        string
	KeyPair            services.KeyPair
	IssuedAt           time.Time
	AccessTokenExpiry  time.Duration
	IdTokenExpiry      time.Duration
	RefreshTokenExpiry time.Duration
	Nonce              string
	AuthenticatedAt    time.Time
}

func (t *TokenGenerationParams) ToAccessTokenGenerationParams() AccessTokenGenerationParams {
	return AccessTokenGenerationParams{
		ExternalUrl:       t.ExternalUrl,
		VirtualServerName: t.VirtualServerName,
		ClientId:          t.ClientId,
		ApplicationId:     t.ApplicationId,
		GrantedScopes:     t.GrantedScopes,
		IssuedAt:          t.IssuedAt,
		Expiry:            t.AccessTokenExpiry,
		UserId:            t.UserId,
		KeyPair:           t.KeyPair,
	}
}

func (t *TokenGenerationParams) ToIdTokenGenerationParams() IdTokenGenerationParams {
	return IdTokenGenerationParams{
		ClientId:          t.ClientId,
		ExternalUrl:       t.ExternalUrl,
		UserDisplayName:   t.UserDisplayName,
		VirtualServerName: t.VirtualServerName,
		GrantedScopes:     t.GrantedScopes,
		Nonce:             t.Nonce,
		IssuedAt:          t.IssuedAt,
		Expiry:            t.IdTokenExpiry,
		UserId:            t.UserId,
		KeyPair:           t.KeyPair,
		AuthenticatedAt:   t.AuthenticatedAt,
	}
}

func (t *TokenGenerationParams) ToRefreshTokenGenerationParams() RefreshTokenGenerationParams {
	return RefreshTokenGenerationParams{
		VirtualServerName: t.VirtualServerName,
		GrantedScopes:     t.GrantedScopes,
		ClientId:          t.ClientId,
		UserId:            t.UserId,
	}
}

type RefreshTokenGenerationParams struct {
	VirtualServerName string
	GrantedScopes     []string
	ClientId          string
	UserId            uuid.UUID
}

type AccessTokenGenerationParams struct {
	ExternalUrl       string
	VirtualServerName string
	ClientId          string
	ApplicationId     uuid.UUID
	GrantedScopes     []string
	IssuedAt          time.Time
	Expiry            time.Duration
	UserId            uuid.UUID
	KeyPair           services.KeyPair
}

type IdTokenGenerationParams struct {
	ClientId          string
	ExternalUrl       string
	UserDisplayName   string
	VirtualServerName string
	Nonce             string
	IssuedAt          time.Time
	Expiry            time.Duration
	UserId            uuid.UUID
	KeyPair           services.KeyPair
	GrantedScopes     []string
	AuthenticatedAt   time.Time
}

type GeneratedTokens struct {
	IdToken      string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

func generateIdToken(params IdTokenGenerationParams) (string, error) {
	kid := params.KeyPair.GetKid()

	jwtSigningMethod, err := getJwtSigningMethod(params.KeyPair.Algorithm())
	if err != nil {
		return "", fmt.Errorf("getting jwt signing method: %w", err)
	}

	idTokenClaims := jwt.MapClaims{
		"sub": params.UserId,
		"iss": fmt.Sprintf("%s/oidc/%s", params.ExternalUrl, params.VirtualServerName),
		"aud": []string{params.ClientId},
		"iat": params.IssuedAt.Unix(),
		"exp": params.IssuedAt.Add(params.Expiry).Unix(),
	}

	if !params.AuthenticatedAt.IsZero() {
		idTokenClaims["auth_time"] = params.AuthenticatedAt.Unix()
	}

	if slices.Contains(params.GrantedScopes, "profile") {
		idTokenClaims["name"] = params.UserDisplayName
	}

	if params.Nonce != "" {
		idTokenClaims["nonce"] = params.Nonce
	}

	idToken := jwt.NewWithClaims(jwtSigningMethod, idTokenClaims)
	idToken.Header["kid"] = kid
	return idToken.SignedString(params.KeyPair.PrivateKey())
}

func generateAccessToken(ctx context.Context, params AccessTokenGenerationParams) (string, error) {
	kid := params.KeyPair.GetKid()

	jwtSigningMethod, err := getJwtSigningMethod(params.KeyPair.Algorithm())
	if err != nil {
		return "", fmt.Errorf("getting jwt signing method: %w", err)
	}

	accessTokenClaims, err := mapClaims(ctx, params)
	if err != nil {
		return "", fmt.Errorf("mapping claims: %w", err)
	}

	accessTokenClaims["sub"] = params.UserId
	accessTokenClaims["iss"] = fmt.Sprintf("%s/oidc/%s", params.ExternalUrl, params.VirtualServerName)
	accessTokenClaims["aud"] = []string{params.ClientId}
	accessTokenClaims["scopes"] = params.GrantedScopes
	accessTokenClaims["iat"] = params.IssuedAt.Unix()
	accessTokenClaims["exp"] = params.IssuedAt.Add(params.Expiry).Unix()

	accessToken := jwt.NewWithClaims(jwtSigningMethod, accessTokenClaims)
	accessToken.Header["kid"] = kid
	accessToken.Header["typ"] = "at+jwt" // RFC 9068
	return accessToken.SignedString(params.KeyPair.PrivateKey())
}

func mapClaims(ctx context.Context, params AccessTokenGenerationParams) (jwt.MapClaims, error) {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Id(params.UserId)
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	userRoleAssignmentRepository := ioc.GetDependency[repositories.UserRoleAssignmentRepository](scope)
	globalUserRoleAssignmentFilter := repositories.NewUserRoleAssignmentFilter().
		// TODO: add virtual server filter
		UserId(params.UserId).
		ApplicationId(nil).
		IncludeRole()
	globalUserRoleAssignments, _, err := userRoleAssignmentRepository.List(ctx, globalUserRoleAssignmentFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user role assignments: %w", err)
	}

	globalRoles := make([]string, 0, len(globalUserRoleAssignments))
	for _, userRoleAssignment := range globalUserRoleAssignments {
		globalRoles = append(globalRoles, userRoleAssignment.RoleInfo().Name)
	}

	applicationUserRoleAssignmentFilter := repositories.NewUserRoleAssignmentFilter().
		// TODO: add virtual server filter
		UserId(params.UserId).
		ApplicationId(&params.ApplicationId).
		IncludeRole()
	applicationUserRoleAssignments, _, err := userRoleAssignmentRepository.List(ctx, applicationUserRoleAssignmentFilter)
	applicationRoles := make([]string, 0, len(applicationUserRoleAssignments))
	if err != nil {
		return nil, fmt.Errorf("getting user role assignments: %w", err)
	}
	for _, userRoleAssignment := range applicationUserRoleAssignments {
		applicationRoles = append(applicationRoles, userRoleAssignment.RoleInfo().Name)
	}

	claimsMapper := ioc.GetDependency[claimsMapping.ClaimsMapper](scope)
	mappedClaims := claimsMapper.MapClaims(
		ctx,
		params.ApplicationId,
		claimsMapping.Params{
			Roles:            globalRoles,
			ApplicationRoles: applicationRoles,
		},
	)

	claims := jwt.MapClaims(mappedClaims)
	return claims, nil
}

func generateRefreshTokenInfo(params RefreshTokenGenerationParams) (string, error) {
	refreshTokenInfo := jsonTypes.NewRefreshTokenInfo(
		params.VirtualServerName,
		params.ClientId,
		params.UserId,
		params.GrantedScopes,
	)
	refreshTokenInfoJson, err := json.Marshal(refreshTokenInfo)
	if err != nil {
		return "", fmt.Errorf("marshaling refresh token info: %w", err)
	}
	return string(refreshTokenInfoJson), nil
}

func generateTokens(ctx context.Context, params TokenGenerationParams, tokenService services.TokenService) (GeneratedTokens, error) {
	idTokenString, err := generateIdToken(params.ToIdTokenGenerationParams())
	if err != nil {
		return GeneratedTokens{}, fmt.Errorf("signing id token: %w", err)
	}

	accessTokenString, err := generateAccessToken(ctx, params.ToAccessTokenGenerationParams())
	if err != nil {
		return GeneratedTokens{}, fmt.Errorf("signing access token: %w", err)
	}

	refreshTokenInfoString, err := generateRefreshTokenInfo(params.ToRefreshTokenGenerationParams())
	if err != nil {
		return GeneratedTokens{}, err
	}

	refreshTokenString, err := tokenService.GenerateAndStoreToken(
		ctx,
		services.OidcRefreshTokenTokenType,
		refreshTokenInfoString,
		params.RefreshTokenExpiry,
	)
	if err != nil {
		return GeneratedTokens{}, fmt.Errorf("generating refresh token: %w", err)
	}

	return GeneratedTokens{
		IdToken:      idTokenString,
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int(params.AccessTokenExpiry.Seconds()),
	}, nil
}

func handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	clientId, clientSecret, hasBasicAuth := r.BasicAuth()
	if !hasBasicAuth {
		clientId = r.Form.Get("client_id")
	}

	_, err := authenticateApplication(ctx, clientId, clientSecret)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("authenticating application: %w", err))
		return
	}

	tokenService := ioc.GetDependency[services.TokenService](scope)
	refreshTokenInfoString, err := tokenService.GetToken(ctx, services.OidcRefreshTokenTokenType, r.Form.Get("refresh_token"))
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting refresh token: %w", err))
		return
	}

	var refreshTokenInfo jsonTypes.RefreshTokenInfo
	err = json.Unmarshal([]byte(refreshTokenInfoString), &refreshTokenInfo)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("unmarshaling refresh token info: %w", err))
		return
	}

	if refreshTokenInfo.ClientId != clientId {
		utils.HandleHttpError(w, fmt.Errorf("invalid client id"))
		return
	}

	err = tokenService.DeleteToken(ctx, services.OidcRefreshTokenTokenType, r.Form.Get("refresh_token"))
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("deleting refresh token: %w", err))
		return
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Id(refreshTokenInfo.UserId)
	user, err := userRepository.First(ctx, userFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting user: %w", err))
		return
	}
	if user == nil {
		utils.HandleHttpError(w, fmt.Errorf("user not found"))
		return
	}

	clockService := ioc.GetDependency[clock.Service](scope)
	now := clockService.Now()

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(refreshTokenInfo.VirtualServerName)
	virtualServer, err := virtualServerRepository.First(r.Context(), virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		Name(clientId)
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting application: %w", err))
		return
	}
	if application == nil {
		utils.HandleHttpError(w, fmt.Errorf("application not found"))
		return
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(refreshTokenInfo.VirtualServerName, virtualServer.SigningAlgorithm())

	tokenDuration := time.Hour // TODO: make this configurable per virtual server

	params := TokenGenerationParams{
		UserId:             refreshTokenInfo.UserId,
		VirtualServerName:  refreshTokenInfo.VirtualServerName,
		ClientId:           clientId,
		ApplicationId:      application.Id(),
		GrantedScopes:      refreshTokenInfo.GrantedScopes,
		UserDisplayName:    user.DisplayName(),
		UserPrimaryEmail:   user.PrimaryEmail(),
		ExternalUrl:        config.C.Server.ExternalUrl,
		KeyPair:            keyPair,
		IssuedAt:           now,
		AccessTokenExpiry:  tokenDuration,
		IdTokenExpiry:      tokenDuration,
		RefreshTokenExpiry: tokenDuration,
	}

	tokens, err := generateTokens(ctx, params, tokenService)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := RefreshTokenResponse{
		TokenType:    "Bearer",
		IdToken:      tokens.IdToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("encoding response: %w", err))
		return
	}
}

type TokenExchangeResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
}

func handleTokenExchange(w http.ResponseWriter, r *http.Request) {
	subjectToken := r.Form.Get("subject_token")
	subjectTokenType := r.Form.Get("subject_token_type")

	if subjectToken == "" {
		utils.HandleHttpError(w, fmt.Errorf("missing subject token"))
		return
	}

	if subjectTokenType == "" {
		utils.HandleHttpError(w, fmt.Errorf("missing subject token type"))
		return
	}

	if subjectTokenType != "urn:ietf:params:oauth:token-type:access_token" {
		utils.HandleHttpError(w, fmt.Errorf("unsupported subject token type: %s", subjectTokenType))
	}

	ctx := r.Context()

	virtualServerName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server name: %w", err))
		return
	}

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](middlewares.GetScope(ctx))
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
	virtualServer, err := virtualServerRepository.First(ctx, virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}
	if virtualServer == nil {
		utils.HandleHttpError(w, fmt.Errorf("virtual server not found"))
		return
	}

	scope := middlewares.GetScope(ctx)

	token, err := jwt.Parse(subjectToken, func(token *jwt.Token) (any, error) {
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid claims")
		}

		subject, err := claims.GetSubject()
		if err != nil {
			return nil, fmt.Errorf("getting subject: %w", err)
		}

		issuer, err := claims.GetIssuer()
		if err != nil {
			return nil, fmt.Errorf("getting issuer: %w", err)
		}

		if issuer != subject {
			return nil, fmt.Errorf("invalid issuer, issuer has to be the same as subject for service account token exchange")
		}

		keyIdClaim, ok := token.Header["kid"]
		if !ok {
			return nil, fmt.Errorf("missing kid header")
		}

		keyIdString, ok := keyIdClaim.(string)
		if !ok {
			return nil, fmt.Errorf("expected kid header to be a string")
		}

		audienceClaim, err := claims.GetAudience()
		if err != nil {
			return nil, fmt.Errorf("getting audience: %w", err)
		}
		if len(audienceClaim) != 1 {
			return nil, fmt.Errorf("expected audience to be a single string")
		}

		scopesClaim, ok := claims["scopes"]
		if !ok {
			return nil, fmt.Errorf("missing scopes claim")
		}

		scopesString, ok := scopesClaim.(string)
		if !ok {
			return nil, fmt.Errorf("expected scopes claim to be a space separated string")
		}

		scopes := strings.Split(scopesString, " ")
		if len(scopes) == 0 {
			return nil, fmt.Errorf("expected scopes claim to be a space separated string")
		}

		if !slices.Contains(scopes, "openid") {
			return nil, fmt.Errorf("expected scopes claim to contain openid")
		}

		// TODO: check if scopes are valid

		userId, err := uuid.Parse(subject)
		if err != nil {
			return nil, fmt.Errorf("parsing subject: %w", err)
		}

		userRepository := ioc.GetDependency[repositories.UserRepository](scope)
		userFilter := repositories.NewUserFilter().
			VirtualServerId(virtualServer.Id()).
			Id(userId)
		user, err := userRepository.First(ctx, userFilter)
		if err != nil {
			return nil, fmt.Errorf("getting user: %w", err)
		}
		if user == nil {
			return nil, fmt.Errorf("user not found")
		}

		if !user.IsServiceUser() {
			return nil, fmt.Errorf("user is not a service user")
		}

		keyId, err := uuid.Parse(keyIdString)
		if err != nil {
			return nil, fmt.Errorf("parsing key id: %w", err)
		}

		credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
		credentialFilter := repositories.NewCredentialFilter().
			Type(repositories.CredentialTypeServiceUserKey).
			UserId(userId).
			Id(keyId)
		credential, err := credentialRepository.First(ctx, credentialFilter)
		if err != nil {
			return nil, fmt.Errorf("getting credential: %w", err)
		}
		if credential == nil {
			return nil, fmt.Errorf("credential not found")
		}

		serviceUserKeyDetails, err := credential.ServiceUserKeyDetails()
		if err != nil {
			return nil, fmt.Errorf("getting service user key details: %w", err)
		}

		return serviceUserKeyDetails.PublicKey, nil
	})
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("parsing subject token: %w", err))
		return
	}
	if !token.Valid {
		utils.HandleHttpError(w, fmt.Errorf("invalid subject token"))
		return
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting subject: %w", err))
		return
	}

	userId, err := uuid.Parse(subject)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("parsing subject: %w", err))
		return
	}

	audience, err := token.Claims.GetAudience()
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting audience: %w", err))
		return
	}
	if len(audience) != 1 {
		utils.HandleHttpError(w, fmt.Errorf("expected audience to be a single string"))
	}

	applicationName := audience[0]

	scopesClaim := token.Claims.(jwt.MapClaims)["scopes"].(string)
	scopes := strings.Split(scopesClaim, " ")

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		Name(applicationName)
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting application: %w", err))
		return
	}
	if application == nil {
		utils.HandleHttpError(w, fmt.Errorf("application not found"))
		return
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(virtualServerName, virtualServer.SigningAlgorithm())

	clockService := ioc.GetDependency[clock.Service](scope)
	now := clockService.Now()

	accessToken, err := generateAccessToken(ctx, AccessTokenGenerationParams{
		UserId:            userId,
		VirtualServerName: virtualServer.Name(),
		ClientId:          applicationName,
		ApplicationId:     application.Id(),
		GrantedScopes:     scopes,
		ExternalUrl:       config.C.Server.ExternalUrl,
		KeyPair:           keyPair,
		IssuedAt:          now,
		Expiry:            time.Minute * 5, // TODO: make this configurable per virtual server
	})
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("generating access token: %w", err))
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")

	response := TokenExchangeResponse{
		AccessToken:     accessToken,
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		TokenType:       "Bearer",
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("encoding response: %w", err))
		return
	}
}
