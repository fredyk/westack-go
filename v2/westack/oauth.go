package westack

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/fredyk/westack-go/v2/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/amazon"
	"golang.org/x/oauth2/bitbucket"
	"golang.org/x/oauth2/cern"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/fitbit"
	"golang.org/x/oauth2/foursquare"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/heroku"
	"golang.org/x/oauth2/hipchat"
	"golang.org/x/oauth2/instagram"
	"golang.org/x/oauth2/kakao"
	"golang.org/x/oauth2/linkedin"
	"golang.org/x/oauth2/mailchimp"
	"golang.org/x/oauth2/mailru"
	"golang.org/x/oauth2/mediamath"
	"golang.org/x/oauth2/microsoft"
	"golang.org/x/oauth2/nokiahealth"
	"golang.org/x/oauth2/odnoklassniki"
	"golang.org/x/oauth2/paypal"
	"golang.org/x/oauth2/slack"
	"golang.org/x/oauth2/spotify"
	"golang.org/x/oauth2/stackoverflow"
	"golang.org/x/oauth2/twitch"
	"golang.org/x/oauth2/uber"
	"golang.org/x/oauth2/vk"
	"golang.org/x/oauth2/yahoo"
	"golang.org/x/oauth2/yandex"
)

var defaultScopes = map[string][]string{
	"google":     {"https://www.googleapis.com/auth/userinfo.email"},
	"amazon":     {"profile"},
	"bitbucket":  {},
	"cern":       {},
	"facebook":   {"email"},
	"fitbit":     {"profile"},
	"foursquare": {},
	// include access to repos
	"github":        {},
	"gitlab":        {"read_user"},
	"heroku":        {},
	"hipchat":       {},
	"instagram":     {"basic"},
	"kakao":         {"profile"},
	"linkedin":      {"r_emailaddress"},
	"mailchimp":     {"profile"},
	"mailru":        {"userinfo"},
	"mediamath":     {},
	"microsoft":     {"User.Read"},
	"nokiahealth":   {"user.info"},
	"odnoklassniki": {"VALUABLE_ACCESS"},
	"paypal":        {"openid"},
	"slack":         {"identity.basic"},
	"spotify":       {"user-read-email"},
	"stackoverflow": {"read_inbox"},
	"twitch":        {"user:read:email"},
	"uber":          {"profile"},
	"vk":            {"email"},
	"yahoo":         {"profile"},
	"yandex":        {"login:email"},
}

var userInfoUrls = map[string]string{
	"google":        "https://www.googleapis.com/oauth2/v3/userinfo",
	"amazon":        "https://api.amazon.com/user/profile",
	"bitbucket":     "https://api.bitbucket.org/2.0/user",
	"cern":          "https://oauth.web.cern.ch/v1/api/profile",
	"facebook":      "https://graph.facebook.com/me?fields=email",
	"fitbit":        "https://api.fitbit.com/1/user/-/profile.json",
	"foursquare":    "https://api.foursquare.com/v2/users/self",
	"github":        "https://api.github.com/user/emails",
	"gitlab":        "https://gitlab.com/api/v4/user",
	"heroku":        "https://api.heroku.com/account",
	"hipchat":       "https://api.hipchat.com/v2/oauth/token",
	"instagram":     "https://api.instagram.com/v1/users/self",
	"kakao":         "https://kapi.kakao.com/v2/user/me",
	"linkedin":      "https://api.linkedin.com/v2/me",
	"mailchimp":     "https://login.mailchimp.com/oauth2/metadata",
	"mailru":        "https://oauth.mail.ru/userinfo",
	"mediamath":     "https://api.mediamath.com/api/v2.0/user",
	"microsoft":     "https://graph.microsoft.com/v1.0/me",
	"nokiahealth":   "https://account.health.nokia.com/v2/user",
	"odnoklassniki": "https://api.ok.ru/fb.do",
	"paypal":        "https://api.paypal.com/v1/identity/openidconnect/userinfo",
	"slack":         "https://slack.com/api/users.identity",
	"spotify":       "https://api.spotify.com/v1/me",
	"stackoverflow": "https://api.stackexchange.com/2.2/me",
	"twitch":        "https://api.twitch.tv/helix/users",
	"uber":          "https://api.uber.com/v1/me",
	"vk":            "https://api.vk.com/method/users.get",
	"yahoo":         "https://api.login.yahoo.com/openid/v1/userinfo",
	"yandex":        "https://login.yandex.ru/info",
}

var knownEndpoints = map[string]oauth2.Endpoint{
	"google":        google.Endpoint,
	"amazon":        amazon.Endpoint,
	"bitbucket":     bitbucket.Endpoint,
	"cern":          cern.Endpoint,
	"facebook":      facebook.Endpoint,
	"fitbit":        fitbit.Endpoint,
	"foursquare":    foursquare.Endpoint,
	"github":        github.Endpoint,
	"gitlab":        gitlab.Endpoint,
	"heroku":        heroku.Endpoint,
	"hipchat":       hipchat.Endpoint,
	"instagram":     instagram.Endpoint,
	"kakao":         kakao.Endpoint,
	"linkedin":      linkedin.Endpoint,
	"mailchimp":     mailchimp.Endpoint,
	"mailru":        mailru.Endpoint,
	"mediamath":     mediamath.Endpoint,
	"microsoft":     microsoft.LiveConnectEndpoint,
	"nokiahealth":   nokiahealth.Endpoint,
	"odnoklassniki": odnoklassniki.Endpoint,
	"paypal":        paypal.Endpoint,
	"slack":         slack.Endpoint,
	"spotify":       spotify.Endpoint,
	"stackoverflow": stackoverflow.Endpoint,
	"twitch":        twitch.Endpoint,
	"uber":          uber.Endpoint,
	"vk":            vk.Endpoint,
	"yahoo":         yahoo.Endpoint,
	"yandex":        yandex.Endpoint,
}

// Store callback URLs in cookies
func storeCallbackUrlsInCookies(ctx *fiber.Ctx, successUrl, failureUrl string) {
	// Validate and sanitize URLs
	cleanSuccessUrl := validateAndSanitizeUrl(successUrl)
	cleanFailureUrl := validateAndSanitizeUrl(failureUrl)

	if cleanSuccessUrl == "" || cleanFailureUrl == "" {
		fmt.Printf("[ERROR] Invalid callback URLs provided - success: %q, failure: %q\n", successUrl, failureUrl)
		return
	}

	// Store in cookies with HttpOnly and secure settings
	ctx.Cookie(&fiber.Cookie{
		Name:     "OAuth_SuccessURL",
		Value:    cleanSuccessUrl,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   3600, // 1 hour
	})

	ctx.Cookie(&fiber.Cookie{
		Name:     "OAuth_FailureURL",
		Value:    cleanFailureUrl,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   3600, // 1 hour
	})

	fmt.Printf("[DEBUG] Stored callback URLs in cookies\n")
	fmt.Printf("[DEBUG] Success URL stored: %v\n", cleanSuccessUrl)
	fmt.Printf("[DEBUG] Failure URL stored: %v\n", cleanFailureUrl)
}

// Retrieve callback URLs from cookies
func getCallbackUrlsFromCookies(ctx *fiber.Ctx) (string, string, bool) {
	successUrl := ctx.Cookies("OAuth_SuccessURL", "")
	failureUrl := ctx.Cookies("OAuth_FailureURL", "")

	if successUrl != "" && failureUrl != "" {
		fmt.Printf("[DEBUG] Retrieved callback URLs from cookies\n")
		fmt.Printf("[DEBUG] Success URL retrieved: %v\n", successUrl)
		fmt.Printf("[DEBUG] Failure URL retrieved: %v\n", failureUrl)
		return successUrl, failureUrl, true
	} else {
		fmt.Printf("[DEBUG] No callback URLs found in cookies\n")
		return "", "", false
	}
}

// Clear callback URL cookies after use
func clearCallbackUrlCookies(ctx *fiber.Ctx) {
	ctx.Cookie(&fiber.Cookie{
		Name:     "OAuth_SuccessURL",
		Value:    "",
		HTTPOnly: true,
		Expires:  time.Now().Add(-time.Hour),
	})

	ctx.Cookie(&fiber.Cookie{
		Name:     "OAuth_FailureURL",
		Value:    "",
		HTTPOnly: true,
		Expires:  time.Now().Add(-time.Hour),
	})

	fmt.Printf("[DEBUG] Cleared callback URL cookies\n")
}

// Validate and sanitize URL to prevent corruption
func validateAndSanitizeUrl(url string) string {
	if url == "" {
		return ""
	}

	// Remove any null bytes or control characters that could cause corruption
	cleaned := strings.ReplaceAll(url, "\x00", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	cleaned = strings.ReplaceAll(cleaned, "\t", "")

	// Trim whitespace
	cleaned = strings.TrimSpace(cleaned)

	fmt.Printf("[DEBUG] URL validation - original: %q, cleaned: %q\n", url, cleaned)
	return cleaned
}

func mountOauthRoutes(app *WeStack, loadedModel *model.StatefulModel, systemContext *model.EventContext) {

	appPublicOrigin := app.Viper.GetString("publicOrigin")
	finalTokenTtl := app.Viper.GetFloat64("ttl")
	if finalTokenTtl <= 0.0 {
		finalTokenTtl = 30 * 86400
	}
	globalSuccessUrl := app.Viper.GetString("oauth2.successRedirect")
	globalFailureUrl := app.Viper.GetString("oauth2.failureRedirect")

	userProviders := app.Viper.GetStringMap("oauth2.providers")
	for providerName, providerConfig := range userProviders {
		provider := providerConfig.(map[string]interface{})
		providerName := providerName

		providerRedirectUri := fmt.Sprintf("%s%s/oauth/%s/callback", appPublicOrigin, loadedModel.BaseUrl, providerName)
		loginPath := fmt.Sprintf("/oauth/%s", providerName)
		fullLoginPath := fmt.Sprintf("%s%s%s", appPublicOrigin, loadedModel.BaseUrl, loginPath)

		scopes := defaultScopes[providerName]
		if v := provider["scopes"]; v != nil {
			if castedScopes, ok := v.([]string); ok {
				scopes = append(scopes, castedScopes...)
			} else if castedScopes, ok := v.([]any); ok {
				for _, scope := range castedScopes {
					scopes = append(scopes, scope.(string))
				}
			} else {
				fmt.Printf("[ERROR] Invalid scopes for provider %v: %T(%v)\n", providerName, v, v)
			}
		}

		var additionalUserInfoMapping map[string]interface{}
		if v := provider["additionaluserinfo"]; v != nil {
			additionalUserInfoMapping = v.(map[string]interface{})
		}

		userInfoUrl := userInfoUrls[providerName]

		endpoint, ok := knownEndpoints[providerName]
		if !ok {
			// lookup for oauth2.providers.%s.authUrl and oauth2.providers.%s.tokenUrl
			authUrl := app.Viper.GetString(fmt.Sprintf("oauth2.providers.%s.authUrl", providerName))
			tokenUrl := app.Viper.GetString(fmt.Sprintf("oauth2.providers.%s.tokenUrl", providerName))
			userInfoUrl = app.Viper.GetString(fmt.Sprintf("oauth2.providers.%s.userInfoUrl", providerName))
			if authUrl != "" && tokenUrl != "" {
				endpoint = oauth2.Endpoint{
					AuthURL:  authUrl,
					TokenURL: tokenUrl,
				}
			} else {
				fmt.Printf("[ERROR] Invalid oauth2 provider: %v (authUrl=%v, tokenUrl=%v)\n", providerName, authUrl, tokenUrl)
				continue
			}
		}

		// oauth2 client
		oauthConfig := &oauth2.Config{
			ClientID:     app.Viper.GetString(fmt.Sprintf("oauth2.providers.%s.clientID", providerName)),
			ClientSecret: app.Viper.GetString(fmt.Sprintf("oauth2.providers.%s.clientSecret", providerName)),
			RedirectURL:  providerRedirectUri,
			Scopes:       scopes,
			Endpoint:     endpoint,
		}

		fmt.Println(`
=========================================
		
  [INFO] ` + providerName + ` Oauth login: ` + fullLoginPath + `
  [INFO] ` + providerName + ` Oauth Redirect URI: ` + providerRedirectUri + `

=========================================
		`)

		// oauth login endpoint
		loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {

			cookie := ""
			if v := eventContext.Ctx.Cookies("SSID"); v != "" {
				cookie = v
				fmt.Printf("[DEBUG] Using existing SSID cookie: %v\n", cookie)
			} else {
				cookie = wst.GenerateCookie()
				eventContext.Ctx.Cookie(&fiber.Cookie{
					Name:  "SSID",
					Value: cookie,
				})
				fmt.Printf("[DEBUG] Generated new SSID cookie: %v\n", cookie)
			}
			successUrl := eventContext.Ctx.Query("success_url", "")
			failureUrl := eventContext.Ctx.Query("failure_url", "")

			fmt.Printf("[DEBUG] Received query params - success_url: %v, failure_url: %v\n", successUrl, failureUrl)

			if successUrl != "" && failureUrl != "" {
				fmt.Printf("[DEBUG] About to store URLs - success: %q, failure: %q\n", successUrl, failureUrl)
				storeCallbackUrlsInCookies(eventContext.Ctx, successUrl, failureUrl)
			} else {
				fmt.Printf("[DEBUG] No callback URLs provided in query params\n")
			}
			oauthStateString := utils.CreateOauthStateString(cookie)
			fmt.Printf("[DEBUG] Oauth state: %v\n", oauthStateString)

			url := oauthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
			fmt.Printf("[DEBUG] Redirect to URL: %v\n", url)
			return eventContext.Ctx.Redirect(url)

		}, model.RemoteMethodOptions{
			Name:        fmt.Sprintf(string(wst.OperationNameOauthLogin), providerName),
			Description: "Logins with Google",
			Http: model.RemoteMethodOptionsHttp{
				Path: loginPath,
				Verb: "get",
			},
		})

		// oauth callback endpoint
		loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {

			cookie := eventContext.Ctx.Cookies("SSID")
			fmt.Printf("[DEBUG] Processing callback for SSID cookie: %v\n", cookie)

			successUrl := globalSuccessUrl
			failureUrl := globalFailureUrl

			overrideSuccessUrl, overrideFailureUrl, hasOverride := getCallbackUrlsFromCookies(eventContext.Ctx)
			if hasOverride {
				fmt.Printf("[DEBUG] Found override callback URLs in cookies\n")
				fmt.Printf("[DEBUG] Override Success URL: %q\n", overrideSuccessUrl)
				fmt.Printf("[DEBUG] Override Failure URL: %q\n", overrideFailureUrl)

				// Additional corruption detection
				if strings.Contains(overrideSuccessUrl, "sometuncothervalueconandothervalue") ||
					strings.Contains(overrideFailureUrl, "sometuncothervalueconandothervalue") {
					fmt.Printf("[ERROR] DETECTED CORRUPTION in callback URLs!\n")
					fmt.Printf("[ERROR] Success URL corrupted: %q\n", overrideSuccessUrl)
					fmt.Printf("[ERROR] Failure URL corrupted: %q\n", overrideFailureUrl)
				}

				// use the overrided URLs
				successUrl = overrideSuccessUrl
				failureUrl = overrideFailureUrl

				// Clear the cookies after use
				clearCallbackUrlCookies(eventContext.Ctx)
			} else {
				fmt.Printf("[DEBUG] Using global callback URLs\n")
				fmt.Printf("[DEBUG] Global Success URL: %v\n", globalSuccessUrl)
				fmt.Printf("[DEBUG] Global Failure URL: %v\n", globalFailureUrl)
			}

			if cookie == "" {
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("missing session"))
			}

			receivedState := eventContext.Query.GetString("state")
			fmt.Printf("[DEBUG] Received state: %v\n", receivedState)
			fmt.Printf("[DEBUG] Verify with SSID: %v\n", cookie)

			if receivedState == "" {
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("missing oauth state"))
			}

			ok = utils.VerifyOauthState(cookie, receivedState)
			if !ok {
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("invalid oauth state"))
			}

			oauthCode := eventContext.Query.GetString("code")
			token, err := oauthConfig.Exchange(eventContext.Ctx.Context(), oauthCode)
			if err != nil {
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("oauth exchange failed: %w", err))
			}

			userInfo, err := oauthConfig.Client(eventContext.Ctx.Context(), token).Get(userInfoUrl)
			if err != nil {
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to get user info: %w", err))
			}

			defer userInfo.Body.Close()

			var infoDataA wst.A
			var userInfoData wst.M
			// github returns an array
			if providerName == "github" {
				err = json.NewDecoder(userInfo.Body).Decode(&infoDataA)
				if err != nil {
					return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to decode user info: %w", err))
				}
				if len(infoDataA) == 0 {
					return verboseRedirect(eventContext, failureUrl, fmt.Errorf("empty user info"))
				}
				userInfoData = infoDataA[0]
				if userInfoData == nil {
					return verboseRedirect(eventContext, failureUrl, fmt.Errorf("empty user info"))
				}
			} else {
				err = json.NewDecoder(userInfo.Body).Decode(&userInfoData)
				if err != nil {
					return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to decode user info: %w", err))
				}
			}

			isEmail := false
			var login string
			if v := userInfoData["email"]; v != nil {
				login = v.(string)
			}
			if !isValidEmail(login) {
				if v := userInfoData["emails"]; v != nil {
					emails := v.([]any)
					if len(emails) > 0 {
						login = emails[0].(string)
					}
				}
				if !isValidEmail(login) {
					if v := userInfoData["primary_email"]; v != nil {
						login = v.(string)
					}
				}
				if !isValidEmail(login) {
					if v := userInfoData["login"]; v != nil && v.(string) != "" {
						login = v.(string)
					} else if v := userInfoData["username"]; v != nil && v.(string) != "" {
						login = v.(string)
					} else if v := userInfoData["nickname"]; v != nil && v.(string) != "" {
						login = v.(string)
					}
				}
			}
			if isValidEmail(login) {
				isEmail = true
			} else {
				fmt.Printf("WARNING: missing email in user info %v", userInfoData)
			}

			var additionalUserInfo wst.M
			if additionalUserInfoMapping != nil {
				additionalUserInfo = wst.M{}
				for key, value := range additionalUserInfoMapping {
					if v := userInfoData[value.(string)]; v != nil {
						additionalUserInfo[key] = v
					} else {
						fmt.Printf("WARNING: Field '%v' was not found in user info %v\n", value, userInfoData)
					}
				}
			}

			// check if userCredentials exists
			userCredentials, err := app.accountCredentialsModel.FindOne(&wst.Filter{
				Where: &wst.Where{
					// "email":    login,
					"$or": []wst.M{
						{"email": login},
						{"username": login},
					},
					"provider": string(ProviderOAuth2Prefix) + providerName,
				},
				Include: &wst.Include{
					{
						Relation: "account",
					},
				},
			}, systemContext)

			if err != nil {
				fmt.Printf("[DEBUG] Error while fetching credentials by email-provider: %v\n", err)
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to fetch oauth credentials: %w", err))
			}

			var convertedAccount *model.StatefulInstance
			var accountId string
			var account model.Instance
			needNewAccount := userCredentials == nil

			if userCredentials != nil {

				account = userCredentials.GetOne("account")

				if account == nil {
					needNewAccount = true
				}
			}

			if needNewAccount {
				// search by password
				userCredentials, err = app.accountCredentialsModel.FindOne(&wst.Filter{
					Where: &wst.Where{
						"$and": []wst.M{
							{
								"$or": []wst.M{
									{"email": login},
									{"username": login},
								},
							},
							{
								"$or": []wst.M{
									{"provider": ProviderPassword},
									{"password": wst.M{"$exists": true}},
								},
							},
						},
					},
					Include: &wst.Include{
						{
							Relation: "account",
						},
					},
				}, systemContext)

				if err != nil {
					fmt.Printf("[DEBUG] Error while fetching credentials by email-password: %v\n", err)
					return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to fetch password credentials: %w", err))
				}

				if userCredentials == nil {

					// create new account
					fmt.Printf("[DEBUG] Creating new account for email: %v\n", login)
					plainAccount := wst.M{
						// "email":         login,
						"emailVerified": true,
						"provider":      string(ProviderOAuth2Prefix) + providerName,
					}
					for key, value := range additionalUserInfo {
						plainAccount[key] = value
					}
					if isEmail {
						plainAccount["email"] = login
					} else {
						plainAccount["username"] = login
					}
					createdAccount, err := loadedModel.Create(plainAccount, systemContext)

					if err != nil {
						fmt.Printf("[DEBUG] Error while creating account: %v\n", err)
						return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to create account: %w", err))
					}

					convertedAccount = createdAccount.(*model.StatefulInstance)
					accountId = convertedAccount.GetString("id")

				} else {

					accountId = userCredentials.GetString("accountId")

					convertedAccount = userCredentials.GetOne("account").(*model.StatefulInstance)

				}

				// create new credentials
				if isEmail {
					fmt.Printf("[DEBUG] Creating new credentials for email: %v\n", login)
				} else {
					fmt.Printf("[DEBUG] Creating new credentials for login: %v\n", login)
				}
				plainCredentials := wst.M{
					"accountId": accountId,
					// "email":        login,
					"provider":     string(ProviderOAuth2Prefix) + providerName,
					"accessToken":  token.AccessToken,
					"refreshToken": token.RefreshToken,
					"expiry":       token.Expiry,
					"tokenType":    token.TokenType,
					"scope":        token.Extra("scope"),
				}
				if isEmail {
					plainCredentials["email"] = login
				} else {
					plainCredentials["username"] = login
				}
				_, err = app.accountCredentialsModel.Create(plainCredentials, systemContext)

				if err != nil {
					fmt.Printf("[DEBUG] Error while creating credentials: %v\n", err)
					return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to create credentials: %w", err))
				}

			} else {

				convertedAccount = account.(*model.StatefulInstance)
				accountId = convertedAccount.GetString("id")

				// update credentials
				fmt.Printf("[DEBUG] Updating credentials for email: %v\n", login)
				_, err = userCredentials.UpdateAttributes(wst.M{
					"accessToken": token.AccessToken,
					"expiry":      token.Expiry,
					"tokenType":   token.TokenType,
					"scope":       token.Extra("scope"),
				}, systemContext)

				if err != nil {
					fmt.Printf("[ERROR] Could not update credentials: %v\n", err)
				}

			}

			roleNames := []string{"USER"}

			roleContext := &model.EventContext{
				BaseContext:            systemContext,
				DisableTypeConversions: true,
			}

			roleEntries, _ := app.roleMappingModel.FindMany(&wst.Filter{Where: &wst.Where{
				"principalType": "USER",
				"$or": []wst.M{
					{
						"principalId": accountId,
					},
					{
						"principalId": convertedAccount.Id,
					},
				},
			}, Include: &wst.Include{{Relation: "role"}}}, roleContext).All()

			for _, roleEntry := range roleEntries {
				role := roleEntry.GetOne("role")
				roleNames = append(roleNames, role.ToJSON()["name"].(string))
			}

			ttl := 30 * 86400.0
			bearer := model.CreateBearer(accountId, float64(time.Now().Unix()), ttl, roleNames)
			// sign the bearer
			jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, bearer.Claims)
			tokenString, err := jwtToken.SignedString(loadedModel.App.JwtSecretKey)
			if err != nil {
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to sign token: %w", err))
			}

			fmt.Printf("[DEBUG] Redirecting to success URL: %v\n", successUrl)
			if strings.Contains(successUrl, "?") {
				successUrl += "&"
			} else {
				successUrl += "?"
			}
			return eventContext.Ctx.Redirect(successUrl + "access_token=" + tokenString)

		}, model.RemoteMethodOptions{
			Name:        fmt.Sprintf(string(wst.OperationNameOauthLoginCallback), providerName),
			Description: fmt.Sprintf("%s OAuth2 callback", providerName),
			Http: model.RemoteMethodOptionsHttp{
				Path: fmt.Sprintf("/oauth/%s/callback", providerName),
				Verb: "get",
			},
		})

		// Get saved access token, only available for github and gitlab
		if providerName == "github" || providerName == "gitlab" {
			loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {

				accountId := eventContext.Bearer.Account.Id

				// check if userCredentials exists
				userCredentials, err := app.accountCredentialsModel.FindOne(&wst.Filter{
					Where: &wst.Where{
						"accountId": accountId,
						"provider":  string(ProviderOAuth2Prefix) + providerName,
					},
				}, systemContext)

				if err != nil {
					fmt.Printf("[DEBUG] Error while fetching credentials by accountId-provider: %v\n", err)
					return wst.CreateError(fiber.ErrInternalServerError, "ERR_INTERNAL_SERVER_ERROR", fiber.Map{"message": fmt.Sprintf("Failed to fetch oauth credentials: %v", err)}, "Error")
				}

				if userCredentials == nil {
					return wst.CreateError(fiber.ErrNotFound, "ERR_CREDENTIALS_NOT_FOUND", fiber.Map{"message": "Credentials not found"}, "Error")
				}

				eventContext.Result = wst.M{
					"tokenType":    userCredentials.GetString("tokenType"),
					"accessToken":  userCredentials.GetString("accessToken"),
					"refreshToken": userCredentials.GetString("refreshToken"),
					"expiry":       userCredentials.GetInt("expiry"),
					"scope":        userCredentials.GetString("scope"),
				}

				return nil

			}, model.RemoteMethodOptions{
				Name:        fmt.Sprintf(string(wst.OperationNameOauthGetOauthCredentials), providerName),
				Description: fmt.Sprintf("Get %s access token", providerName),
				Http: model.RemoteMethodOptionsHttp{
					Verb: "get",
					Path: fmt.Sprintf("/oauth/%s/credentials", providerName),
				},
			})

		}
	}

}
