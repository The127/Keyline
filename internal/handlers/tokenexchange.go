package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/services"
	"github.com/The127/Keyline/utils"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const exchangedAccessTokenLifetime = time.Hour

type TokenExchangeResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
}

func handleTokenExchange(w http.ResponseWriter, r *http.Request) {
	subjectToken := r.Form.Get("subject_token")
	if subjectToken == "" {
		writeOAuthError(w, "invalid_request", "subject_token is required")
		return
	}

	subjectTokenType := r.Form.Get("subject_token_type")
	if subjectTokenType != "urn:ietf:params:oauth:token-type:access_token" {
		writeOAuthError(w, "invalid_request", "subject_token_type must be urn:ietf:params:oauth:token-type:access_token")
		return
	}

	audienceName := r.Form.Get("audience")
	if audienceName == "" {
		writeOAuthError(w, "invalid_request", "audience is required")
		return
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server name: %w", err))
		return
	}

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrNil(ctx, virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}
	if virtualServer == nil {
		utils.HandleHttpError(w, fmt.Errorf("virtual server not found"))
		return
	}

	credentials := readClientCredentials(r)
	if credentials.clientId == "" && !credentials.hasAssertion() {
		writeOAuthError(w, "invalid_client", "client authentication is required")
		return
	}

	exchanger, err := authenticateApplication(ctx, virtualServer, credentials)
	if err != nil {
		writeOAuthError(w, "invalid_client", "client authentication failed")
		return
	}
	if exchanger.Type() != repositories.ApplicationTypeConfidential {
		writeOAuthError(w, "invalid_client", "client authentication failed")
		return
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	clockService := ioc.GetDependency[clock.Service](scope)
	now := clockService.Now()
	issuer := fmt.Sprintf("%s/oidc/%s", config.C.Server.ExternalUrl, virtualServer.Name())

	subjectJwt, err := jwt.Parse(subjectToken, func(token *jwt.Token) (any, error) {
		keyPair, err := keyService.GetKey(virtualServer.Name(), config.SigningAlgorithm(token.Method.Alg()))
		if err != nil {
			return nil, fmt.Errorf("getting key: %w", err)
		}
		return keyPair.PublicKey(), nil
	},
		jwt.WithIssuer(issuer),
		jwt.WithAudience(exchanger.Name()),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		writeOAuthError(w, "invalid_request", "subject_token is invalid")
		return
	}
	if subjectJwt.Header["typ"] != exchanger.AccessTokenHeaderType() {
		writeOAuthError(w, "invalid_request", "subject_token is invalid")
		return
	}

	subjectClaims := subjectJwt.Claims.(jwt.MapClaims)
	if _, ok := subjectClaims["scopes"]; !ok {
		writeOAuthError(w, "invalid_request", "subject_token is invalid")
		return
	}

	subjectExpiry, err := subjectClaims.GetExpirationTime()
	if err != nil || subjectExpiry == nil {
		writeOAuthError(w, "invalid_request", "subject_token is invalid")
		return
	}

	subject, err := subjectClaims.GetSubject()
	if err != nil {
		writeOAuthError(w, "invalid_request", "subject_token is invalid")
		return
	}
	userId, err := uuid.Parse(subject)
	if err != nil {
		writeOAuthError(w, "invalid_request", "subject_token is invalid")
		return
	}

	userFilter := repositories.NewUserFilter().Id(userId).VirtualServerId(virtualServer.Id())
	user, err := dbContext.Users().FirstOrNil(ctx, userFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting user: %w", err))
		return
	}
	if user == nil {
		writeOAuthError(w, "invalid_request", "subject_token is invalid")
		return
	}

	scopes, err := extractScopes(subjectJwt)
	if err != nil {
		writeOAuthError(w, "invalid_request", "subject_token is invalid")
		return
	}

	target, err := findApplication(ctx, virtualServer, audienceName)
	if err != nil {
		writeOAuthError(w, "invalid_target", "audience is not available to this client")
		return
	}
	if !target.TrustsExchanger(exchanger.Name()) {
		writeOAuthError(w, "invalid_target", "audience is not available to this client")
		return
	}

	expiry := min(exchangedAccessTokenLifetime, subjectExpiry.Sub(now))

	actor := map[string]any{"sub": exchanger.Name()}
	if previousActor, ok := subjectClaims["act"]; ok {
		actor["act"] = previousActor
	}

	keyPair, err := keyService.GetKey(virtualServer.Name(), appSigningAlgorithm(virtualServer, target))
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	accessToken, err := generateAccessToken(ctx, AccessTokenGenerationParams{
		UserId:                user.Id(),
		VirtualServerName:     virtualServer.Name(),
		ClientId:              target.Name(),
		ApplicationId:         target.Id(),
		GrantedScopes:         scopes,
		ExternalUrl:           config.C.Server.ExternalUrl,
		KeyPair:               keyPair,
		IssuedAt:              now,
		Expiry:                expiry,
		HeaderType:            target.AccessTokenHeaderType(),
		UserinfoInAccessToken: target.UserinfoInAccessToken(),
		Actor:                 actor,
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
		ExpiresIn:       int(expiry.Seconds()),
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("encoding response: %w", err))
		return
	}
}
