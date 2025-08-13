package handlers

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"Keyline/utils"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
