package handlers

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"net/http"
	"slices"
	"time"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"
	"github.com/golang-jwt/jwt/v5"
)

const (
	clientAssertionTypeJwtBearer = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	maxClientAssertionLifetime   = 5 * time.Minute
)

var clientAssertionSigningMethods = []string{string(config.SigningAlgorithmRS256), string(config.SigningAlgorithmEdDSA)}

type clientCredentials struct {
	clientId            string
	clientSecret        string
	clientAssertion     string
	clientAssertionType string
}

func readClientCredentials(r *http.Request) clientCredentials {
	credentials := clientCredentials{
		clientAssertion:     r.Form.Get("client_assertion"),
		clientAssertionType: r.Form.Get("client_assertion_type"),
	}

	clientId, clientSecret, hasBasicAuth := r.BasicAuth()
	if hasBasicAuth {
		credentials.clientId = clientId
		credentials.clientSecret = clientSecret
	} else {
		credentials.clientId = r.Form.Get("client_id")
		credentials.clientSecret = r.Form.Get("client_secret")
	}

	return credentials
}

func (c clientCredentials) hasAssertion() bool {
	return c.clientAssertion != "" || c.clientAssertionType != ""
}

func findApplication(ctx context.Context, virtualServer *repositories.VirtualServer, name string) (*repositories.Application, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		Name(name)
	application, err := dbContext.Applications().FirstOrNil(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}
	if application == nil {
		return nil, fmt.Errorf("application not found")
	}

	return application, nil
}

// authenticateApplication looks up an application by name within the given
// virtual server and verifies client authentication.
//
// Authentication rules:
//   - Confidential clients MUST present a non-empty client_secret that matches
//     the stored hash. Empty or wrong secret -> error.
//   - Confidential clients registered with private_key_jwt MUST present a
//     client_assertion signed by one of their registered keys (RFC 7523) and
//     MUST NOT send a client_secret.
//   - Public clients MUST NOT send a client_secret (they have none registered).
//     They authenticate the redemption via PKCE, which is checked separately
//     by the caller against the bound code.
//
// The virtual server scoping closes a tenant-isolation bug: previously the
// lookup was by name only, which would let a confidential client in tenant A
// authenticate against a token request bound to tenant B (when names collided).
func authenticateApplication(
	ctx context.Context,
	virtualServer *repositories.VirtualServer,
	credentials clientCredentials,
) (*repositories.Application, error) {
	if credentials.hasAssertion() {
		return authenticateApplicationWithAssertion(ctx, virtualServer, credentials)
	}

	application, err := findApplication(ctx, virtualServer, credentials.clientId)
	if err != nil {
		return nil, err
	}

	switch application.Type() {
	case repositories.ApplicationTypeConfidential:
		if application.AuthenticatesWith(repositories.TokenEndpointAuthMethodPrivateKeyJwt) {
			return nil, fmt.Errorf("client_assertion is required for this client")
		}
		if credentials.clientSecret == "" {
			return nil, fmt.Errorf("client_secret is required for confidential clients")
		}
		if !utils.CheapCompareHash(credentials.clientSecret, application.HashedSecret()) {
			return nil, fmt.Errorf("invalid secret")
		}
		return application, nil

	case repositories.ApplicationTypePublic:
		if credentials.clientSecret != "" {
			return nil, fmt.Errorf("public clients must not present a client_secret")
		}
		return application, nil

	default:
		return nil, fmt.Errorf("unsupported application type: %s", application.Type())
	}
}

// verifyPKCE checks an RFC 7636 PKCE code_verifier against the stored
// code_challenge and method. Only S256 is accepted; "plain" is rejected since
// it provides no protection against an attacker who can read the request.

func authenticateApplicationWithAssertion(
	ctx context.Context,
	virtualServer *repositories.VirtualServer,
	credentials clientCredentials,
) (*repositories.Application, error) {
	if credentials.clientAssertionType != clientAssertionTypeJwtBearer {
		return nil, fmt.Errorf("unsupported client_assertion_type")
	}
	if credentials.clientAssertion == "" {
		return nil, fmt.Errorf("client_assertion is required")
	}
	if credentials.clientSecret != "" {
		return nil, fmt.Errorf("client_secret must not be combined with client_assertion")
	}

	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)
	clockService := ioc.GetDependency[clock.Service](scope)
	now := clockService.Now()

	issuer := fmt.Sprintf("%s/oidc/%s", config.C.Server.ExternalUrl, virtualServer.Name())
	tokenEndpoint := issuer + "/token"

	var application *repositories.Application
	token, err := jwt.Parse(credentials.clientAssertion, func(token *jwt.Token) (any, error) {
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid claims")
		}

		subject, err := claims.GetSubject()
		if err != nil || subject == "" {
			return nil, fmt.Errorf("client_assertion has no sub")
		}

		tokenIssuer, err := claims.GetIssuer()
		if err != nil || tokenIssuer != subject {
			return nil, fmt.Errorf("client_assertion iss and sub must both be the client_id")
		}

		if credentials.clientId != "" && credentials.clientId != subject {
			return nil, fmt.Errorf("client_id does not match client_assertion")
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("client_assertion has no kid header")
		}

		found, err := findApplication(ctx, virtualServer, subject)
		if err != nil {
			return nil, err
		}
		if !found.AuthenticatesWith(repositories.TokenEndpointAuthMethodPrivateKeyJwt) {
			return nil, fmt.Errorf("client does not authenticate with private_key_jwt")
		}

		applicationKeyFilter := repositories.NewApplicationKeyFilter().
			ApplicationId(found.Id()).
			Kid(kid)
		applicationKey, err := dbContext.ApplicationKeys().FirstOrNil(ctx, applicationKeyFilter)
		if err != nil {
			return nil, fmt.Errorf("getting application key: %w", err)
		}
		if applicationKey == nil {
			return nil, fmt.Errorf("unknown kid")
		}

		publicKey, err := utils.ParsePublicKeyPem(applicationKey.PublicKey())
		if err != nil {
			return nil, fmt.Errorf("parsing application key: %w", err)
		}

		application = found
		return publicKey, nil
	},
		jwt.WithValidMethods(clientAssertionSigningMethods),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return nil, fmt.Errorf("client_assertion is invalid: %w", err)
	}
	if !token.Valid || application == nil {
		return nil, fmt.Errorf("client_assertion is invalid")
	}

	claims := token.Claims.(jwt.MapClaims)

	audiences, err := claims.GetAudience()
	if err != nil {
		return nil, fmt.Errorf("client_assertion has no aud")
	}
	if !slices.Contains(audiences, issuer) && !slices.Contains(audiences, tokenEndpoint) {
		return nil, fmt.Errorf("client_assertion aud must be the issuer or the token endpoint")
	}

	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		return nil, fmt.Errorf("client_assertion has no exp")
	}
	if expiresAt.After(now.Add(maxClientAssertionLifetime)) {
		return nil, fmt.Errorf("client_assertion exp is more than %s in the future", maxClientAssertionLifetime)
	}

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return nil, fmt.Errorf("client_assertion has no jti")
	}

	return application, nil
}
