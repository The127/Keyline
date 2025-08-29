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
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
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
		Issuer: fmt.Sprintf("%s/virtual-servers/%s", config.C.Server.ExternalUrl, vsName),

		AuthorizationEndpoint: fmt.Sprintf("%s/virtual-servers/%s/authorize", config.C.Server.ExternalUrl, vsName),
		TokenEndpoint:         fmt.Sprintf("%s/virtual-servers/%s/token", config.C.Server.ExternalUrl, vsName),
		UserinfoEndpoint:      "todo", // TODO:
		JwksUri:               fmt.Sprintf("%s/virtual-servers/%s/.well-known/jwks.json", config.C.Server.ExternalUrl, vsName),

		ResponseTypesSupported:           []string{"code"}, // TODO: maybe support more
		SubjectTypesSupported:            []string{"public"},
		IdTokenSigningAlgValuesSupported: []string{"EdDSA"},

		ScopesSupported: []string{"oidc", "email", "profile"}, // TODO: get from db
		ClaimsSupported: []string{"sub", "name", "email"},     // TODO: get from db
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

	// get the virtual server
	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	// get form data (auth request)
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

	// validate the auth request
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

	// TODO: add pkce challenge

	// TODO: get claims from scopes

	now := time.Now() // TODO: add clock service for testing/mocking

	keyService := ioc.GetDependency[services.KeyService](scope)
	keyPair := keyService.GetKey(codeInfo.VirtualServerName)
	key := keyPair.PrivateKey()

	idTokenClaims := jwt.MapClaims{
		"sub":   codeInfo.UserId,
		"iss":   config.C.Server.ExternalUrl,
		"aud":   clientId,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(), // TODO: make this configurable per virtual server
		"name":  user.DisplayName(),
		"email": user.PrimaryEmail(),
	}

	idToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, idTokenClaims)
	idTokenString, err := idToken.SignedString(key)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("signing id token: %w", err))
		return
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"sub":    codeInfo.UserId,
		"scopes": codeInfo.GrantedScopes,
		"iat":    now.Unix(),
		"exp":    now.Add(time.Hour).Unix(), // TODO: make this configurable per virtual server
	})
	accessTokenString, err := accessToken.SignedString(key)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("signing access token: %w", err))
		return
	}

	refreshTokenInfo := jsonTypes.RefreshTokenInfo{
		// TODO: populate with information as we need
	}
	refreshTokenInfoJson, err := json.Marshal(refreshTokenInfo)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("marshaling refresh token info: %w", err))
		return
	}

	refreshTokenString, err := tokenService.GenerateAndStoreToken(
		ctx,
		services.OidcRefreshTokenTokenType,
		string(refreshTokenInfoJson),
		time.Hour, // TODO: make this configurable per virtual server
	)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("generating refresh token: %w", err))
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	scopeString := strings.Join(codeInfo.GrantedScopes, " ")
	response := CodeFlowResponse{
		TokenType:    "Bearer",
		IdToken:      idTokenString,
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		Scope:        scopeString,
		ExpiresIn:    int(time.Hour.Seconds()), // TODO: make this configurable per virtual server
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("encoding response: %w", err))
		return
	}
}

func handleRefreshToken(w http.ResponseWriter, r *http.Request) {

	// TODO: implement

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
