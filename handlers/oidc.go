package handlers

import (
	"Keyline/config"
	"Keyline/ioc"
	"Keyline/jsonTypes"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"Keyline/utils"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
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

type Ed25519JWK struct {
	Kty string `json:"kty"` // Key Type
	Crv string `json:"crv"` // Curve
	Alg string `json:"alg"` // Algorithm
	Use string `json:"use"` // Use (sig = signature)
	Kid string `json:"kid"` // Key ID
	X   string `json:"x"`   // Public key (base64url)
}

type JwksResponseDto struct {
	Keys []Ed25519JWK `json:"keys"`
}

func WellKnownJwks(w http.ResponseWriter, r *http.Request) {
	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(r.Context())
	keyService := ioc.GetDependency[services.KeyService](scope)

	keyPair := keyService.GetKey(vsName)

	kid := computeKID(keyPair.PublicKey())

	jwk := Ed25519JWK{
		Kty: "OKP",
		Crv: "Ed25519",
		Alg: "EdDSA",
		Use: "sig",
		Kid: kid,
		X:   base64.RawURLEncoding.EncodeToString(keyPair.PublicKey()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(JwksResponseDto{
		Keys: []Ed25519JWK{jwk},
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

func computeKID(pub ed25519.PublicKey) string {
	hash := sha256.Sum256(pub)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

type OpenIdConfigurationResponseDto struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserinfoEndpoint                 string   `json:"userinfo_endpoint"`
	EndSessionEndpoint               string   `json:"end_session_endpoint"`
	JwksUri                          string   `json:"jwks_uri"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IdTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
	ClaimsSupported                  []string `json:"claims_supported"`
}

func WellKnownOpenIdConfiguration(w http.ResponseWriter, r *http.Request) {
	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	responseDto := OpenIdConfigurationResponseDto{
		Issuer: fmt.Sprintf("%s/oidc/%s", config.C.Server.ExternalUrl, vsName),

		AuthorizationEndpoint: fmt.Sprintf("%s/oidc/%s/authorize", config.C.Server.ExternalUrl, vsName),
		TokenEndpoint:         fmt.Sprintf("%s/oidc/%s/token", config.C.Server.ExternalUrl, vsName),
		UserinfoEndpoint:      fmt.Sprintf("%s/oidc/%s/userinfo", config.C.Server.ExternalUrl, vsName),
		EndSessionEndpoint:    fmt.Sprintf("%s/oidc/%s/end_session", config.C.Server.ExternalUrl, vsName),
		JwksUri:               fmt.Sprintf("%s/oidc/%s/.well-known/jwks.json", config.C.Server.ExternalUrl, vsName),

		ResponseTypesSupported:           []string{"code"}, // TODO: maybe support more
		SubjectTypesSupported:            []string{"public"},
		IdTokenSigningAlgValuesSupported: []string{"EdDSA"},

		ScopesSupported: []string{"openid", "email", "profile"}, // TODO: get from db
		ClaimsSupported: []string{"sub", "name", "email"},       // TODO: get from db
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(responseDto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type AuthorizationRequest struct {
	ResponseTypes       []string
	VirtualServerName   string
	ApplicationName     string
	RedirectUri         string
	Scopes              []string
	State               string
	ResponseMode        string
	PKCEChallenge       string
	PKCEChallengeMethod string
}

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

	authRequest := AuthorizationRequest{
		ResponseTypes:       strings.Split(r.Form.Get("response_type"), " "),
		VirtualServerName:   vsName,
		ApplicationName:     r.Form.Get("client_id"),
		RedirectUri:         r.Form.Get("redirect_uri"),
		Scopes:              strings.Split(r.Form.Get("scope"), " "),
		State:               r.Form.Get("state"),
		ResponseMode:        r.Form.Get("response_mode"),
		PKCEChallenge:       r.Form.Get("code_challenge"),
		PKCEChallengeMethod: r.Form.Get("code_challenge_method"),
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
		utils.HandleHttpError(w, fmt.Errorf("unsupported response type: %s", authRequest.ResponseTypes[0]))
		return
	}

	if !slices.Contains(authRequest.Scopes, "oidc") {
		utils.HandleHttpError(w, fmt.Errorf("required oidc scope missing"))
		return
	}

	// TODO: check the scopes for email and profile

	tokenService := ioc.GetDependency[services.TokenService](scope)

	s, ok := middlewares.GetSession(ctx)
	if ok {
		// TODO: consent page

		codeInfo := jsonTypes.NewCodeInfo(
			virtualServer.Name(),
			[]string{"email", "openid", "sub"},
			s.UserId(),
		)

		codeInfoString, err := json.Marshal(codeInfo)
		if err != nil {
			utils.HandleHttpError(w, fmt.Errorf("marshaling code info: %w", err))
			return
		}

		code, err := tokenService.GenerateAndStoreToken(ctx, services.OidcCodeTokenType, string(codeInfoString), time.Second*30)
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

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(vsName)
	key := keyPair.PrivateKey()

	idTokenString := r.Form.Get("id_token_hint")
	if idTokenString == "" {
		utils.HandleHttpError(w, fmt.Errorf("id token hint not found"))
		return
	}

	idToken, err := jwt.Parse(idTokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is EdDSA
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key.Public(), nil
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
	clientId := idTokenClaims["aud"].(string)
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

type OidcUserInfoResponseDto struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func OidcUserinfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(vsName)

	err = r.ParseForm()
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	bearer := r.Header.Get("Authorization")
	if bearer == "" {
		utils.HandleHttpError(w, fmt.Errorf("authorization header not found"))
		return
	}

	if !strings.HasPrefix(bearer, "Bearer ") {
		utils.HandleHttpError(w, fmt.Errorf("authorization header is not a bearer token"))
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
		Sub:   userId.String(),
		Email: user.PrimaryEmail(),
		Name:  user.DisplayName(),
	}

	err = json.NewEncoder(w).Encode(tempResult)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

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
	if utils.CheapCompareHash(hashedSecret, applicationSecret) {
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

func handleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	code := r.Form.Get("code")

	tokenService := ioc.GetDependency[services.TokenService](scope)
	valueString, err := tokenService.GetToken(ctx, services.OidcCodeTokenType, code)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting token: %w", err))
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

	now := time.Now() // TODO: add clock service for testing/mocking

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(codeInfo.VirtualServerName)

	tokenDuration := time.Hour // TODO: make this configurable per virtual server

	params := TokenGenerationParams{
		UserId:             codeInfo.UserId,
		VirtualServerName:  codeInfo.VirtualServerName,
		ClientId:           clientId,
		GrantedScopes:      codeInfo.GrantedScopes,
		UserDisplayName:    user.DisplayName(),
		UserPrimaryEmail:   user.PrimaryEmail(),
		ExternalUrl:        config.C.Server.ExternalUrl,
		PrivateKey:         keyPair.PrivateKey(),
		PublicKey:          keyPair.PublicKey(),
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
	GrantedScopes      []string
	UserDisplayName    string
	UserPrimaryEmail   string
	ExternalUrl        string
	PrivateKey         ed25519.PrivateKey
	PublicKey          ed25519.PublicKey
	IssuedAt           time.Time
	AccessTokenExpiry  time.Duration
	IdTokenExpiry      time.Duration
	RefreshTokenExpiry time.Duration
}

type GeneratedTokens struct {
	IdToken      string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

func generateIdToken(params TokenGenerationParams) (string, error) {
	kid := computeKID(params.PublicKey)

	idTokenClaims := jwt.MapClaims{
		"sub":   params.UserId,
		"iss":   fmt.Sprintf("%s/oidc/%s", params.ExternalUrl, params.VirtualServerName),
		"aud":   params.ClientId,
		"iat":   params.IssuedAt.Unix(),
		"exp":   params.IssuedAt.Add(params.IdTokenExpiry).Unix(),
		"name":  params.UserDisplayName,
		"email": params.UserPrimaryEmail,
	}

	idToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, idTokenClaims)
	idToken.Header["kid"] = kid
	return idToken.SignedString(params.PrivateKey)
}

func generateAccessToken(params TokenGenerationParams) (string, error) {
	kid := computeKID(params.PublicKey)

	accessTokenClaims := jwt.MapClaims{
		"sub":    params.UserId,
		"iss":    fmt.Sprintf("%s/oidc/%s", params.ExternalUrl, params.VirtualServerName),
		"scopes": params.GrantedScopes,
		"iat":    params.IssuedAt.Unix(),
		"exp":    params.IssuedAt.Add(params.AccessTokenExpiry).Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, accessTokenClaims)
	accessToken.Header["kid"] = kid
	return accessToken.SignedString(params.PrivateKey)
}

func generateRefreshTokenInfo(params TokenGenerationParams) (string, error) {
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

func generateTokens(ctx context.Context, params TokenGenerationParams, tokenService services.TokenService) (*GeneratedTokens, error) {
	idTokenString, err := generateIdToken(params)
	if err != nil {
		return nil, fmt.Errorf("signing id token: %w", err)
	}

	accessTokenString, err := generateAccessToken(params)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	refreshTokenInfoString, err := generateRefreshTokenInfo(params)
	if err != nil {
		return nil, err
	}

	refreshTokenString, err := tokenService.GenerateAndStoreToken(
		ctx,
		services.OidcRefreshTokenTokenType,
		refreshTokenInfoString,
		params.RefreshTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	return &GeneratedTokens{
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

	now := time.Now() // TODO: add clock service for testing/mocking

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(refreshTokenInfo.VirtualServerName)

	tokenDuration := time.Hour // TODO: make this configurable per virtual server

	params := TokenGenerationParams{
		UserId:             refreshTokenInfo.UserId,
		VirtualServerName:  refreshTokenInfo.VirtualServerName,
		ClientId:           clientId,
		GrantedScopes:      refreshTokenInfo.GrantedScopes,
		UserDisplayName:    user.DisplayName(),
		UserPrimaryEmail:   user.PrimaryEmail(),
		ExternalUrl:        config.C.Server.ExternalUrl,
		PrivateKey:         keyPair.PrivateKey(),
		PublicKey:          keyPair.PublicKey(),
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
