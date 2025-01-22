package westack

import (
	"encoding/json"
	"fmt"
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
	"google":        {"https://www.googleapis.com/auth/userinfo.email"},
	"amazon":        {"profile"},
	"bitbucket":     {},
	"cern":          {},
	"facebook":      {"email"},
	"fitbit":        {"profile"},
	"foursquare":    {},
	"github":        {"user:email"},
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
	"github":        "https://api.github.com/user",
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

func mountOauthRoutes(app *WeStack, loadedModel *model.StatefulModel, systemContext *model.EventContext) {

	appPublicOrigin := app.Viper.GetString("publicOrigin")
	finalTokenTtl := app.Viper.GetFloat64("ttl")
	if finalTokenTtl <= 0.0 {
		finalTokenTtl = 30 * 86400
	}
	successUrl := app.Viper.GetString("oauth2.successRedirect")
	failureUrl := app.Viper.GetString("oauth2.failureRedirect")

	userProviders := app.Viper.GetStringMap("oauth2.providers")
	for providerName, providerConfig := range userProviders {
		provider := providerConfig.(map[string]interface{})
		providerName := providerName

		providerRedirectUri := fmt.Sprintf("%s%s/oauth/%s/callback", appPublicOrigin, loadedModel.BaseUrl, providerName)
		loginPath := fmt.Sprintf("/oauth/%s", providerName)
		fullLoginPath := fmt.Sprintf("%s%s%s", appPublicOrigin, loadedModel.BaseUrl, loginPath)

		scopes := defaultScopes[providerName]
		if v := provider["scopes"]; v != nil {
			scopes = append(scopes, v.([]string)...)
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
			} else {
				cookie = wst.GenerateCookie()
				eventContext.Ctx.Cookie(&fiber.Cookie{
					Name:  "SSID",
					Value: cookie,
				})
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

			if cookie == "" {
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("missing session"))
			}

			receivedState := eventContext.Query.GetString("state")
			fmt.Printf("[DEBUG] Received state: %v\n", receivedState)
			fmt.Printf("[DEBUG] Verify with SSID: %v\n", cookie)

			if receivedState == "" {
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("missing oauth state"))
			}

			ok := utils.VerifyOauthState(cookie, receivedState)
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

			type userInfoResponse struct {
				Email string `json:"email"`
			}

			var userInfoData userInfoResponse
			err = json.NewDecoder(userInfo.Body).Decode(&userInfoData)
			if err != nil {
				return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to decode user info: %w", err))
			}

			// check if userCredentials exists
			userCredentials, err := app.accountCredentialsModel.FindOne(&wst.Filter{
				Where: &wst.Where{
					"email":    userInfoData.Email,
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

			var account *model.StatefulInstance
			var accountId string

			if userCredentials == nil {
				// search by password
				userCredentials, err = app.accountCredentialsModel.FindOne(&wst.Filter{
					Where: &wst.Where{
						"email": userInfoData.Email,
						"$or": []wst.M{
							{"provider": ProviderPassword},
							{"password": wst.M{"$exists": true}},
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
					fmt.Printf("[DEBUG] Creating new account for email: %v\n", userInfoData.Email)
					createdAccount, err := loadedModel.Create(wst.M{
						"email":         userInfoData.Email,
						"emailVerified": true,
						"provider":      string(ProviderOAuth2Prefix) + providerName,
					}, systemContext)

					if err != nil {
						fmt.Printf("[DEBUG] Error while creating account: %v\n", err)
						return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to create account: %w", err))
					}

					account = createdAccount.(*model.StatefulInstance)
					accountId = account.GetString("id")

				} else {

					accountId = userCredentials.GetString("accountId")

					account = userCredentials.GetOne("account").(*model.StatefulInstance)

				}

				// create new credentials
				fmt.Printf("[DEBUG] Creating new credentials for email: %v\n", userInfoData.Email)
				_, err = app.accountCredentialsModel.Create(wst.M{
					"accountId":    accountId,
					"email":        userInfoData.Email,
					"provider":     string(ProviderOAuth2Prefix) + providerName,
					"accessToken":  token.AccessToken,
					"refreshToken": token.RefreshToken,
					"expiry":       token.Expiry,
					"tokenType":    token.TokenType,
					"scope":        token.Extra("scope"),
				}, systemContext)

				if err != nil {
					fmt.Printf("[DEBUG] Error while creating credentials: %v\n", err)
					return verboseRedirect(eventContext, failureUrl, fmt.Errorf("failed to create credentials: %w", err))
				}

			} else {
				account = userCredentials.GetOne("account").(*model.StatefulInstance)
				accountId = account.GetString("id")

				// update credentials
				fmt.Printf("[DEBUG] Updating credentials for email: %v\n", userInfoData.Email)
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
						"principalId": account.Id,
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

			return eventContext.Ctx.Redirect(successUrl + "?access_token=" + tokenString)

		}, model.RemoteMethodOptions{
			Name:        fmt.Sprintf(string(wst.OperationNameOauthLoginCallback), providerName),
			Description: fmt.Sprintf("%s OAuth2 callback", providerName),
			Http: model.RemoteMethodOptionsHttp{
				Path: fmt.Sprintf("/oauth/%s/callback", providerName),
				Verb: "get",
			},
		})
	}

}
