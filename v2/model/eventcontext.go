package model

import (
	"fmt"
	"strings"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"

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
	ExecutionId            string // Unique ID per execution flow to isolate queued operations
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

	// X-Api-Key authentication: if no user from JWT, try X-Api-Key header
	if user == nil {
		apiKey := string(c.Request().Header.Peek("X-Api-Key"))
		if apiKey != "" {
			apiKeyModelRaw, err := loadedModel.App.FindModel("ApiKey")
			if err == nil {
				apiKeyModel := apiKeyModelRaw.(*StatefulModel)
				systemCtx := &EventContext{Bearer: &BearerToken{Account: &BearerAccount{System: true}}}
				apiKeyInstance, err := apiKeyModel.FindOne(&wst.Filter{
					Where: &wst.Where{
						"key":     apiKey,
						"enabled": true,
					},
				}, systemCtx)
				if err == nil && apiKeyInstance != nil {
					keyId := apiKeyInstance.GetID()
					var apiRoles []BearerRole
					// Las propiedades (no relaciones) se leen del JSON: Get() es solo para relaciones
					// y devolvería nil para "roles". Desde Mongo el array llega como primitive.A;
					// in-memory puede ser []interface{}. Manejamos ambos.
					var rolesList []interface{}
					switch v := apiKeyInstance.ToJSON()["roles"].(type) {
					case primitive.A:
						rolesList = v
					case []interface{}:
						rolesList = v
					}
					for _, r := range rolesList {
						if roleName, ok := r.(string); ok {
							apiRoles = append(apiRoles, BearerRole{Name: roleName})
						}
					}
					return &BearerToken{
						Account: &BearerAccount{
							Id:   keyId,
							Data: apiKeyInstance.ToJSON(),
						},
						Roles:   apiRoles,
						Claims:  bearerClaims,
						Raw:     apiKey,
					}, nil
				}
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
