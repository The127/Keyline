package handlers

import (
	"Keyline/config"
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"Keyline/utils"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
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
	err = json.NewEncoder(w).Encode(JwksResponseDto{
		Keys: []Ed25519JWK{jwk},
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func computeKID(pub ed25519.PublicKey) string {
	hash := sha256.Sum256(pub)
	return base64.RawURLEncoding.EncodeToString(hash[:8])
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

		AuthorizationEndpoint: "todo", // TODO:
		TokenEndpoint:         "todo", // TODO:
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
