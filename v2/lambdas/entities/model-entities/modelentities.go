package modelentities

import (
	"fmt"

	"github.com/goccy/go-json"

	"github.com/fredyk/westack-go/v2/lambdas/entities"
	"github.com/golang-jwt/jwt"
)

type Config struct {
	Name        string                `json:"name"`
	Plural      string                `json:"plural"`
	Base        string                `json:"base"`
	Public      bool                  `json:"public"`
	Properties  map[string]Property   `json:"properties"`
	Relations   *map[string]*Relation `json:"relations"`
	Hidden      []string              `json:"hidden"`
	Protected   []string              `json:"protected"`
	Validations []Validation          `json:"validations"`
	Casbin      CasbinConfig          `json:"casbin"`
	Cache       CacheConfig           `json:"cache"`
}

type Property struct {
	Type     interface{} `json:"type"`
	Required bool        `json:"required"`
	Default  interface{} `json:"default"`
}

type Relation struct {
	Type       string  `json:"type"`
	Model      string  `json:"model"`
	PrimaryKey *string `json:"primaryKey"`
	ForeignKey *string `json:"foreignKey"`
	Options    struct {
		//Inverse bool `json:"inverse"`
		SkipAuth bool `json:"skipAuth"`
	} `json:"options"`
}

type ACL struct {
	AccessType    string `json:"accessType"`
	PrincipalType string `json:"principalType"`
	PrincipalId   string `json:"principalId"`
	Permission    string `json:"permission"`
	Property      string `json:"property"`
}

type CasbinConfig struct {
	RequestDefinition  string   `json:"requestDefinition"`
	PolicyDefinition   string   `json:"policyDefinition"`
	RoleDefinition     string   `json:"roleDefinition"`
	PolicyEffect       string   `json:"policyEffect"`
	MatchersDefinition string   `json:"matchersDefinition"`
	Policies           []string `json:"policies"`
}

type CacheConfig struct {
	Datasource    string     `json:"datasource"`
	Ttl           int        `json:"ttl"`
	Keys          [][]string `json:"keys"`
	ExcludeFields []string   `json:"excludeFields"`
}

type Validation struct {
	If         map[string]Condition  `json:"if"`
	Then       *Validation           `json:"then"`
	AllOf      []Validation          `json:"allOf"`
	OneOf      []Validation          `json:"oneOf"`
	Properties map[string]Validation `json:"properties"`
	NotEmpty   bool                  `json:"notEmpty"`
}

type Condition struct {
	Equals      interface{}   `json:"equals"`
	NotEquals   interface{}   `json:"notEquals"`
	Contains    []interface{} `json:"contains"`
	NotContains []interface{} `json:"notContains"`
	Exists      bool          `json:"exists"`
	NotExists   bool          `json:"notExists"`
	Empty       bool          `json:"empty"`
	NotEmpty    bool          `json:"notEmpty"`
}

type HttpContext interface {
}

type EventContext struct {
	Bearer                 *BearerToken
	BaseContext            *EventContext
	Remote                 *RemoteMethodOptions
	Filter                 *entities.Filter
	Data                   *entities.M
	Query                  *entities.M
	Instance               Instance
	Ctx                    HttpContext
	IsNewInstance          bool
	Result                 interface{}
	Model                  Model
	ModelID                interface{}
	StatusCode             int
	DisableTypeConversions bool
	SkipFieldProtection    bool
	OperationName          string
	OperationId            int64
	Handled                bool
}

type BearerToken struct {
	Account *BearerAccount
	Roles   []BearerRole
	Raw     string
	Claims  jwt.MapClaims
}

type RemoteMethodOptionsHttp struct {
	Path string
	Verb string
}

type ArgHttp struct {
	Source string
}

type RemoteMethodOptionsHttpArg struct {
	Arg         string
	Type        string
	Description string
	Http        ArgHttp
	Required    bool
}

type RemoteMethodOptionsHttpArgs []RemoteMethodOptionsHttpArg

type RemoteMethodOptions struct {
	Name        string
	Description string
	Accepts     RemoteMethodOptionsHttpArgs
	Http        RemoteMethodOptionsHttp
}

type BearerAccount struct {
	Id     interface{}
	Data   interface{}
	System bool
}

type BearerRole struct {
	Name string
}

type StatusCode int

const (
	// HttpCodeOK is the HTTP status code for OK.
	StatusOK StatusCode = 200
	// HttpCodeBadRequest is the HTTP status code for Bad Request.
	StatusBadRequest StatusCode = 400

	// HttpCodeUnauthorized is the HTTP status code for Unauthorized.
	StatusUnauthorized StatusCode = 401

	// HttpCodeForbidden is the HTTP status code for Forbidden.
	StatusForbidden StatusCode = 403

	// HttpCodeNotFound is the HTTP status code for Not Found.
	StatusNotFound StatusCode = 404

	// HttpCodeInternalServerError is the HTTP status code for Internal Server Error.
	StatusInternalServerError StatusCode = 500
)

type HttpError struct {
	Code    int
	Message string
}

var (
	// Error codes
	ErrNotFound HttpError = HttpError{
		Code:    int(StatusNotFound),
		Message: "not found",
	}
	ErrBadRequest HttpError = HttpError{
		Code:    int(StatusBadRequest),
		Message: "bad request",
	}
	ErrUnauthorized HttpError = HttpError{
		Code:    int(StatusUnauthorized),
		Message: "unauthorized",
	}
	ErrForbidden HttpError = HttpError{
		Code:    int(StatusForbidden),
		Message: "forbidden",
	}
	ErrInternalServerError HttpError = HttpError{
		Code:    int(StatusInternalServerError),
		Message: "internal server error",
	}
)

func (err *HttpError) Error() string {
	return fmt.Sprintf("%d: %s", err.Code, err.Message)
}

func NewError(code int, message string) *HttpError {
	return &HttpError{
		Code:    code,
		Message: message,
	}
}

func CreateError(fiberError *HttpError, code string, details entities.M, name string) *WeStackError {
	return &WeStackError{
		FiberError: fiberError,
		Code:       code,
		Details:    details,
		Name:       name,
	}
}

type WeStackError struct {
	FiberError *HttpError
	Code       string
	Details    entities.M
	Name       string
	detailsSt  *string
}

func (err *WeStackError) Error() string {
	if err.detailsSt == nil {
		bytes, err2 := json.Marshal(err.Details)
		st := ""
		if err2 != nil {
			st = fmt.Sprintf("%v", err.Details)
		} else {
			st = string(bytes)
		}
		err.detailsSt = &st
	}
	return fmt.Sprintf("%v %v: %v", err.FiberError.Code, err.FiberError.Error(), *err.detailsSt)
}
