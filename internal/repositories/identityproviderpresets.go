package repositories

var identityProviderPresets = map[string]IdentityProviderSettings{
	"github": {
		AuthorizationEndpoint: "https://github.com/login/oauth/authorize",
		TokenEndpoint:         "https://github.com/login/oauth/access_token",
		UserinfoEndpoint:      "https://api.github.com/user",
		Scopes:                []string{"read:user", "user:email"},
	},
}

func IdentityProviderPreset(name string) (IdentityProviderSettings, bool) {
	preset, ok := identityProviderPresets[name]
	return preset, ok
}
