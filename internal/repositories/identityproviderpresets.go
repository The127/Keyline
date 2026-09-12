package repositories

var identityProviderPresets = map[string]IdentityProviderSettings{
	"github": {
		AuthorizationEndpoint: "https://github.com/login/oauth/authorize",
		TokenEndpoint:         "https://github.com/login/oauth/access_token",
		UserinfoEndpoint:      "https://api.github.com/user",
		Scopes:                []string{"read:user", "user:email"},
	},
	"google": {
		Issuer:                "https://accounts.google.com",
		AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint:         "https://oauth2.googleapis.com/token",
		UserinfoEndpoint:      "https://openidconnect.googleapis.com/v1/userinfo",
		Scopes:                []string{"openid", "email", "profile"},
	},
}

func IdentityProviderPreset(name string) (IdentityProviderSettings, bool) {
	preset, ok := identityProviderPresets[name]
	return preset, ok
}
