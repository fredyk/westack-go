package westack

type AuthProvider string

var (
	ProviderPassword     AuthProvider = "password"
	ProviderOAuth2Prefix AuthProvider = "oauth2_"
)
