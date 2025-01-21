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
	"golang.org/x/oauth2/google"
)

func mountOauthRoutes(app *WeStack, loadedModel *model.StatefulModel, systemContext *model.EventContext) {

	appPublicOrigin := app.Viper.GetString("publicOrigin")
	googleRedirectUri := fmt.Sprintf("%s%s/oauth/google/callback", appPublicOrigin, loadedModel.BaseUrl)
	const loginPath = "/oauth/google"
	fullLoginPath := fmt.Sprintf("%s%s%s", appPublicOrigin, loadedModel.BaseUrl, loginPath)
	successUrl := app.Viper.GetString("oauth2.successRedirect")
	failureUrl := app.Viper.GetString("oauth2.failureRedirect")
	finalTokenTtl := app.Viper.GetFloat64("ttl")
	if finalTokenTtl <= 0.0 {
		finalTokenTtl = 30 * 86400
	}

	scopes := []string{"https://www.googleapis.com/auth/userinfo.email"}
	if v := app.Viper.GetStringSlice("oauth2.google.scopes"); v != nil {
		scopes = append(scopes, v...)
	}

	// oauth2 client
	googleOauthConfig := &oauth2.Config{
		ClientID:     app.Viper.GetString("oauth2.google.clientID"),
		ClientSecret: app.Viper.GetString("oauth2.google.clientSecret"),
		RedirectURL:  googleRedirectUri,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}

	fmt.Println(`
	=========================================
	
	[INFO] Google Oauth login: ` + fullLoginPath + `
	[INFO] Google Oauth Redirect URI: ` + googleRedirectUri + `

	=========================================
	`)

	// google login endpoint
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

		url := googleOauthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
		return eventContext.Ctx.Redirect(url)

	}, model.RemoteMethodOptions{
		Name:        string(wst.OperationNameGoogleLogin),
		Description: "Logins with Google",
		Http: model.RemoteMethodOptionsHttp{
			Path: loginPath,
			Verb: "get",
		},
	})

	// google callback endpoint
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
		token, err := googleOauthConfig.Exchange(eventContext.Ctx.Context(), oauthCode)
		if err != nil {
			return verboseRedirect(eventContext, failureUrl, fmt.Errorf("oauth exchange failed: %w", err))
		}

		userInfo, err := googleOauthConfig.Client(eventContext.Ctx.Context(), token).Get("https://www.googleapis.com/oauth2/v3/userinfo")
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
				"provider": string(ProviderGoogleOAuth2),
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
					"provider":      string(ProviderGoogleOAuth2),
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
				"provider":     string(ProviderGoogleOAuth2),
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
		Name:        string(wst.OperationNameGoogleLoginCallback),
		Description: "Google OAuth2 callback",
		Http: model.RemoteMethodOptionsHttp{
			Path: "/oauth/google/callback",
			Verb: "get",
		},
	})

}
