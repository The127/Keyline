package jsonTypes

type IdentityProviderLogin struct {
	LoginToken   string `json:"loginToken"`
	ProviderName string `json:"providerName"`
	CodeVerifier string `json:"codeVerifier"`
	Nonce        string `json:"nonce"`
	BrowserToken string `json:"browserToken"`
}
