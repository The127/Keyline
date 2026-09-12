package identityproviders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/The127/Keyline/internal/repositories"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
)

const (
	requestTimeout  = 10 * time.Second
	maxResponseSize = 1 << 20
)

type Tokens struct {
	AccessToken string
	IdToken     string
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return fmt.Errorf("provider redirected to %s", req.URL)
			},
		},
	}
}

func (c *Client) Exchange(ctx context.Context, settings repositories.IdentityProviderSettings, code string, redirectUri string, codeVerifier string) (Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectUri)
	form.Set("client_id", settings.ClientId)
	form.Set("client_secret", settings.ClientSecret)
	form.Set("code_verifier", codeVerifier)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, settings.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, fmt.Errorf("creating token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	var response struct {
		AccessToken      string `json:"access_token"`
		IdToken          string `json:"id_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	status, err := c.doJson(request, &response)
	if err != nil {
		return Tokens{}, fmt.Errorf("exchanging code: %w", err)
	}
	if response.Error != "" {
		return Tokens{}, fmt.Errorf("token endpoint refused the code: %s %s", response.Error, response.ErrorDescription)
	}
	if status != http.StatusOK {
		return Tokens{}, fmt.Errorf("token endpoint answered %d", status)
	}
	if response.AccessToken == "" {
		return Tokens{}, fmt.Errorf("token endpoint returned no access token")
	}

	return Tokens{
		AccessToken: response.AccessToken,
		IdToken:     response.IdToken,
	}, nil
}

func (c *Client) Userinfo(ctx context.Context, settings repositories.IdentityProviderSettings, accessToken string) (map[string]any, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, settings.UserinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating userinfo request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")

	var claims map[string]any
	status, err := c.doJson(request, &claims)
	if err != nil {
		return nil, fmt.Errorf("fetching userinfo: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint answered %d", status)
	}

	return claims, nil
}

func (c *Client) VerifyIdToken(ctx context.Context, settings repositories.IdentityProviderSettings, idToken string, nonce string) (jwt.MapClaims, error) {
	keySet, err := c.keySet(ctx, settings.Issuer)
	if err != nil {
		return nil, err
	}

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(idToken, claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		keys := keySet.Key(kid)
		if len(keys) == 0 {
			return nil, fmt.Errorf("id token signed with unknown key %q", kid)
		}
		return keys[0].Key, nil
	},
		jwt.WithJSONNumber(),
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "PS256", "PS384", "PS512", "EdDSA"}),
		jwt.WithIssuer(settings.Issuer),
		jwt.WithAudience(settings.ClientId),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return nil, fmt.Errorf("verifying id token: %w", err)
	}

	tokenNonce, _ := claims["nonce"].(string)
	if tokenNonce != nonce {
		return nil, fmt.Errorf("id token nonce does not match the login")
	}

	return claims, nil
}

func (c *Client) keySet(ctx context.Context, issuer string) (*jose.JSONWebKeySet, error) {
	discoveryRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(issuer, "/")+"/.well-known/openid-configuration", nil)
	if err != nil {
		return nil, fmt.Errorf("creating discovery request: %w", err)
	}

	var discovery struct {
		Issuer  string `json:"issuer"`
		JwksUri string `json:"jwks_uri"`
	}
	status, err := c.doJson(discoveryRequest, &discovery)
	if err != nil {
		return nil, fmt.Errorf("fetching discovery document: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("discovery document of %s answered %d", issuer, status)
	}
	if discovery.Issuer != issuer {
		return nil, fmt.Errorf("discovery document of %s names issuer %s", issuer, discovery.Issuer)
	}
	if discovery.JwksUri == "" {
		return nil, fmt.Errorf("discovery document of %s has no jwks_uri", issuer)
	}
	issuerUrl, err := url.Parse(issuer)
	if err != nil {
		return nil, fmt.Errorf("parsing issuer: %w", err)
	}
	jwksUrl, err := url.Parse(discovery.JwksUri)
	if err != nil || jwksUrl.Scheme != issuerUrl.Scheme || jwksUrl.Host == "" {
		return nil, fmt.Errorf("jwks_uri %s of %s is not on the issuer's scheme", discovery.JwksUri, issuer)
	}

	jwksRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.JwksUri, nil)
	if err != nil {
		return nil, fmt.Errorf("creating jwks request: %w", err)
	}

	var keySet jose.JSONWebKeySet
	status, err = c.doJson(jwksRequest, &keySet)
	if err != nil {
		return nil, fmt.Errorf("fetching jwks: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("jwks of %s answered %d", issuer, status)
	}

	return &keySet, nil
}

func (c *Client) doJson(request *http.Request, target any) (int, error) {
	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize))
	if err != nil {
		return response.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	err = decoder.Decode(target)
	if err != nil {
		return response.StatusCode, fmt.Errorf("decoding response with status %d: %w", response.StatusCode, err)
	}

	return response.StatusCode, nil
}

func (c *Client) GithubVerifiedEmail(ctx context.Context, settings repositories.IdentityProviderSettings, accessToken string) (string, bool, error) {
	emailsUrl, err := url.Parse(settings.UserinfoEndpoint)
	if err != nil {
		return "", false, fmt.Errorf("parsing userinfo endpoint: %w", err)
	}

	emailsUrl.Path = strings.TrimSuffix(emailsUrl.Path, "/") + "/emails"
	emailsUrl.RawQuery = ""

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, emailsUrl.String(), nil)
	if err != nil {
		return "", false, fmt.Errorf("creating emails request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	status, err := c.doJson(request, &emails)
	if err != nil {
		return "", false, fmt.Errorf("fetching emails: %w", err)
	}

	if status != http.StatusOK {
		return "", false, fmt.Errorf("emails endpoint answered %d", status)
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, true, nil
		}
	}

	return "", false, nil
}
