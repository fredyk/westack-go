package model

import (
	"fmt"
	"strings"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"

	wst "github.com/fredyk/westack-go/v2/common"
)

type EventContext struct {
	Bearer                 *BearerToken
	BaseContext            *EventContext
	Remote                 *RemoteMethodOptions
	Filter                 *wst.Filter
	Data                   *wst.M
	Query                  *wst.M
	Instance               *StatefulInstance
	Ctx                    *fiber.Ctx
	Ephemeral              *EphemeralData
	IsNewInstance          bool
	Result                 interface{}
	Model                  *StatefulModel
	ModelID                interface{}
	StatusCode             int
	DisableTypeConversions bool
	SkipFieldProtection    bool
	OperationName          wst.OperationName
	OperationId            int64
	Handled                bool
}

func (eventContext *EventContext) UpdateEphemeral(newData *wst.M) {
	if eventContext != nil && newData != nil {
		if eventContext.Ephemeral == nil {
			eventContext.Ephemeral = &EphemeralData{}
		}
		for k, v := range *newData {
			(*eventContext.Ephemeral)[k] = v
		}
	}
}

func (eventContext *EventContext) GetBearer(loadedModel *StatefulModel) (*BearerToken, error) {

	if eventContext.Bearer != nil {
		return eventContext.Bearer, nil
	}
	c := eventContext.Ctx
	authBytes := c.Request().Header.Peek("Authorization")
	authSt := string(authBytes)
	if authSt == "" {
		authSt = c.Query("access_token")
		if authSt != "" {
			authSt = "Bearer " + authSt
		}
	}
	authBearerPair := strings.Split(strings.TrimSpace(authSt), "Bearer ")

	var user *BearerAccount
	roles := make([]BearerRole, 0)
	bearerClaims := jwt.MapClaims{}
	rawToken := ""
	if len(authBearerPair) == 2 && authBearerPair[1] != "" {

		rawToken = authBearerPair[1]

		token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {

			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return loadedModel.App.JwtSecretKey, nil
		})

		if err != nil {
			fmt.Printf("[DEBUG] Invalid token: %s\n", err.Error())
		} else if token != nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				bearerClaims = claims
				claimRoles := claims["roles"]
				userId := claims["accountId"]
				user = &BearerAccount{
					Id:   userId,
					Data: claims,
				}
				if claimRoles != nil {
					for _, role := range claimRoles.([]interface{}) {
						roles = append(roles, BearerRole{
							Name: role.(string),
						})
					}
				}
			} else {
				fmt.Printf("[DEBUG] Invalid token: %s\n", err)
			}
		}

	}
	return &BearerToken{
		Account: user,
		Roles:   roles,
		Claims:  bearerClaims,
		Raw:     rawToken,
	}, nil

}

func (eventContext *EventContext) QueueOperation(operation string, fn func(nextCtx *EventContext) error) {
	eventContext.Model.QueueOperation(operation, eventContext, fn)
}
