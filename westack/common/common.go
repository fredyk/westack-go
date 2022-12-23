package wst

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type M map[string]interface{}

func (m M) GetM(key string) M {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		if vv, ok := v.(M); ok {
			return vv
		} else if vv, ok := v.(map[string]interface{}); ok {
			var out M = make(M, len(vv))
			for k, v := range vv {
				out[k] = v
			}
			return out
		}
	}
	return nil
}

func (m M) GetString(key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if vv, ok := v.(string); ok {
			return vv
		}
	}
	return ""
}

type A []M

func AFromGenericSlice(in *[]interface{}) *A {

	if in == nil {
		return nil
	}

	out := make(A, len(*in))

	for idx, inDoc := range *in {
		if vv, ok := inDoc.(M); ok {
			out[idx] = vv
		} else {
			log.Println("ERROR: AFromGenericSlice: not an M")
			out[idx] = M{}
		}
	}

	return &out
}

func AFromPrimitiveSlice(in *primitive.A) *A {

	if in == nil {
		return nil
	}

	out := make(A, len(*in))

	for idx, inDoc := range *in {
		if vv, ok := inDoc.(primitive.M); ok {

			out[idx] = M{}
			for k, v := range vv {
				out[idx][k] = v
			}

		} else if vv, ok := inDoc.(M); ok {
			out[idx] = vv
		} else {
			log.Println("ERROR: AFromPrimitiveSlice: not a primitive.M")
			out[idx] = M{}
		}
	}

	return &out
}

type Where M

type IncludeItem struct {
	Relation string  `json:"relation"`
	Scope    *Filter `json:"scope"`
}

type Include []IncludeItem
type Order []string

type Filter struct {
	Where   *Where   `json:"where"`
	Include *Include `json:"include"`
	Order   *Order   `json:"order"`
	Skip    int64    `json:"skip"`
	Limit   int64    `json:"limit"`
}

type Stats struct {
	BuildsByModel map[string]map[string]float64
}

type IApp struct {
	Debug          bool
	SwaggerPaths   func() *map[string]M
	FindModel      func(modelName string) (interface{}, error)
	FindDatasource func(datasource string) (interface{}, error)
	JwtSecretKey   []byte
	Viper          *viper.Viper
}

var RegexpIdEntire = regexp.MustCompile("^([0-9a-f]{24})$")
var RegexpIpStart = regexp.MustCompile("^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}")

var regexpTimeZoneReplacing = regexp.MustCompile("([+\\-]\\d{2}):(\\d{2})$")
var regexpDate1 = regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})([+\\-:0-9]+)$")
var regexpDate2 = regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3})([+\\-:0-9]+)$")
var regexpDate3 = regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})([Z]+)?$")
var regexpDate4 = regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3})([Z]+)?$")

type WeStackError struct {
	FiberError *fiber.Error
	Code       string
	Details    fiber.Map
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

func LoadFile(filePath string, out interface{}) error {
	jsonFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	err2 := json.Unmarshal(jsonFile, &out)
	if err2 != nil {
		return err2
	}
	return nil
}

func DashedCase(st string) string {
	var res = strings.ToLower(st[:1])
	compile, err := regexp.Compile("([A-Z])")
	if err != nil {
		panic(err)
	}
	res += string(compile.ReplaceAllFunc([]byte(st[1:]), func(bytes []byte) []byte {
		return []byte("-" + strings.ToLower(string(bytes[0])))
	}))
	return res
}

func CopyMap(src M) M {
	targetMap := make(M)
	for key, value := range src {
		targetMap[key] = value
	}
	return targetMap
}

func Transform(in interface{}, out interface{}) error {
	// TODO: move marshal and unmarshal to easyjson
	bytes, err := bson.Marshal(in)
	if err != nil {
		return err
	}
	err2 := bson.Unmarshal(bytes, out)
	if err2 != nil {
		return err2
	}
	return nil
}

func ParseDate(data string) (time.Time, error) {
	var parsedDate time.Time
	var err error
	if IsDate1(data) {
		layout := "2006-01-02T15:04:05-0700"
		parsedDate, err = time.Parse(layout, regexpTimeZoneReplacing.ReplaceAllString(data, "$1$2"))
	} else if IsDate2(data) {
		layout := "2006-01-02T15:04:05.000-0700"
		parsedDate, err = time.Parse(layout, regexpTimeZoneReplacing.ReplaceAllString(data, "$1$2"))
	} else if IsDate3(data) {
		layout := "2006-01-02T15:04:05Z"
		parsedDate, err = time.Parse(layout, data)
	} else if IsDate4(data) {
		layout := "2006-01-02T15:04:05.000Z"
		parsedDate, err = time.Parse(layout, data)
	}
	if err != nil {
		return time.Time{}, err
	}
	return parsedDate, err
}

func IsAnyDate(data string) bool {
	return IsDate1(data) || IsDate2(data) || IsDate3(data) || IsDate4(data)
}

func IsDate1(data string) bool {
	return regexpDate1.MatchString(data)
}

func IsDate2(data string) bool {
	return regexpDate2.MatchString(data)
}

func IsDate3(data string) bool {
	return regexpDate3.MatchString(data)
}

func IsDate4(data string) bool {
	return regexpDate4.MatchString(data)
}

func CreateError(fiberError *fiber.Error, code string, details fiber.Map, name string) *WeStackError {
	return &WeStackError{
		FiberError: fiberError,
		Code:       code,
		Details:    details,
		Name:       name,
	}
}
