package westack

type AuthProvider string

var (
	ProviderPassword     AuthProvider = "password"
	ProviderGoogleOAuth2 AuthProvider = "google_oauth2"
)
