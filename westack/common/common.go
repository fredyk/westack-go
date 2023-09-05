package wst

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fredyk/westack-go/westack/lib/swaggerhelperinterface"
	"github.com/mailru/easyjson/jlexer"

	"github.com/goccy/go-json"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var NilBytes = []byte{'n', 'u', 'l', 'l'}

type M map[string]interface{}

func (m M) GetM(key string) M {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		if vv, ok := v.(M); ok {
			return vv
		} else if vv, ok := v.(map[string]interface{}); ok {
			var out = make(M, len(vv))
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

func (m M) MarshalEasyJSON(w *jwriter.Writer) {
	if m == nil {
		w.Raw(NilBytes, nil)
		return
	}
	w.RawByte('{')
	first := true
	for k, v := range m {
		if first {
			first = false
		} else {
			w.RawByte(',')
		}
		w.String(k)
		w.RawByte(':')
		switch v := v.(type) {
		case nil:
			w.Raw(NilBytes, nil)
		case bool:
			w.Bool(v)
		case string:
			w.String(v)
		case int:
			w.Int(v)
		case int8:
			w.Int8(v)
		case int16:
			w.Int16(v)
		case int32:
			w.Int32(v)
		case int64:
			w.Int64(v)
		case uint:
			w.Uint(v)
		case uint8:
			w.Uint8(v)
		case uint16:
			w.Uint16(v)
		case uint32:
			w.Uint32(v)
		case uint64:
			w.Uint64(v)
		case float32:
			w.Float32(v)
		case float64:
			w.Float64(v)
		case time.Time:
			w.String(v.Format(time.RFC3339))
		case primitive.ObjectID:
			w.String(v.Hex())
		case primitive.DateTime:
			// format in time.RFC3339Nano v.(primitive.DateTime).Time()
			w.String(v.Time().Format(time.RFC3339Nano))
		default:
			if vv, ok := v.(easyjson.Marshaler); ok {
				//fmt.Printf("Found easyjson.Marshaler: %v at %v\n", vv, k)
				vv.MarshalEasyJSON(w)
			} else if vv, ok := v.(json.Marshaler); ok {
				//fmt.Printf("Found json.Marshaler: %v at %v\n", vv, k)
				bytes, err := vv.MarshalJSON()
				w.Raw(bytes, err)
			} else {
				//fmt.Printf("Found unknown: %v at %v\n", v, k)
				bytes, err := json.Marshal(v)
				w.Raw(bytes, err)
			}
		}
	}
	w.RawByte('}')
}

func (m *M) UnmarshalEasyJSON(l *jlexer.Lexer) {
	if l.IsNull() {
		l.Skip()
		return
	}
	if m == nil {
		*m = make(M)
	}
	inputBytes := l.Raw()
	err := json.Unmarshal(inputBytes, &m)
	if err != nil {
		l.AddError(err)
		return
	}
}

type A []M

func (a A) MarshalEasyJSON(w *jwriter.Writer) {
	if a == nil {
		w.Raw(NilBytes, nil)
		return
	}
	w.RawByte('[')
	first := true
	for _, v := range a {
		if first {
			first = false
		} else {
			w.RawByte(',')
		}
		bytes, err := easyjson.Marshal(v)
		w.Raw(bytes, err)
	}
	w.RawByte(']')
}

func (a A) UnmarshalEasyJSON(l *jlexer.Lexer) {
	if l.IsNull() {
		l.Skip()
		return
	}
	inputBytes := l.Raw()
	err := json.Unmarshal(inputBytes, &a)
	if err != nil {
		l.AddError(err)
		return
	}
}

//func (a A) String() string {
//	if a == nil {
//		return ""
//	}
//	bytes, err := easyjson.Marshal(a)
//	if err != nil {
//		return ""
//	}
//	return string(bytes)
//}

type OperationName string

const (
	OperationNameFindById         OperationName = "findById"
	OperationNameFindMany         OperationName = "findMany"
	OperationNameCount            OperationName = "count"
	OperationNameCreate           OperationName = "create"
	OperationNameUpdateAttributes OperationName = "updateAttributes"
	OperationNameUpdateById       OperationName = "updateById"
	OperationNameUpdateMany       OperationName = "updateMany"
	OperationNameDeleteById       OperationName = "deleteById"
	OperationNameDeleteMany       OperationName = "deleteMany"
)

var (
	NilMap          M = M{"<wst.NilMap>": 1}
	DashedCaseRegex   = regexp.MustCompile("([A-Z])")
)

// AFromGenericSlice converts a generic slice of M to a *A
// This is used to convert the result of a query to a *A
// Returns nil if in is nil
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

// AFromPrimitiveSlice converts a primitive slice of primivite.M or M to a *A
// This is used to convert the result of a query to a *A
// Returns nil if in is nil
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
type AggregationStage M

type IncludeItem struct {
	Relation string  `json:"relation"`
	Scope    *Filter `json:"scope"`
}

type Include []IncludeItem
type Order []string

type Filter struct {
	Where       *Where             `json:"where"`
	Include     *Include           `json:"include"`
	Order       *Order             `json:"order"`
	Skip        int64              `json:"skip"`
	Limit       int64              `json:"limit"`
	Aggregation []AggregationStage `json:"aggregation"`
}

type Stats struct {
	BuildsByModel map[string]map[string]float64
}

type BsonOptions struct {
	Registry *bsoncodec.Registry
}

type IApp struct {
	Debug          bool
	SwaggerHelper  func() swaggerhelperinterface.SwaggerHelper
	FindModel      func(modelName string) (interface{}, error)
	FindDatasource func(datasource string) (interface{}, error)
	JwtSecretKey   []byte
	Viper          *viper.Viper
	Bson           BsonOptions
}

var RegexpIdEntire = regexp.MustCompile("^([0-9a-f]{24})$")
var RegexpIpStart = regexp.MustCompile("^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}")

var regexpTrimMilliseconds = regexp.MustCompile(`^(.*\.\d{3})\d+(.*)$`)
var regexpTimeZoneReplacing = regexp.MustCompile("([+\\-]\\d{2}):(\\d{2})$")
var regexpDate1 = regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})([+\\-:0-9]+)$")
var regexpDate2 = regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3})([+\\-:0-9]+)$")
var regexpDate3 = regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})(Z)?$")
var regexpDate4 = regexp.MustCompile("^(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3})(Z)?$")

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
	jsonFile, err := os.ReadFile(filePath)
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
	var res = strings.ToLower(st[:1]) + string(DashedCaseRegex.ReplaceAllFunc([]byte(st[1:]), func(bytes []byte) []byte {
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

// High-cost operation...
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
		// Trim input up to 3 digits of milliseconds using regex
		// This is needed because time.Parse() does not support more than 3 digits of milliseconds

		prevData := data
		data = regexpTrimMilliseconds.ReplaceAllString(data, "$1$2")
		if prevData != data {
			log.Printf("WARNING: ParseDate: trimming input to 3 digits of milliseconds: %v -> %v", prevData, data)
		}
		parsedDate, err = time.Parse(layout, regexpTimeZoneReplacing.ReplaceAllString(data, "$1$2"))
	} else if is3, groups := IsDate3(data); is3 {
		//layout := "2006-01-02T15:04:05Z"
		var layout string
		isZ := groups[2] == "Z"
		if isZ {
			layout = "2006-01-02T15:04:05Z"
		} else {
			layout = "2006-01-02T15:04:05"
		}
		parsedDate, err = time.Parse(layout, data)
		if !isZ && err == nil && parsedDate.Unix() != 0 {
			parsedDate = parsedDate.In(time.UTC)
		}
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
	isDate3, _ := IsDate3(data)
	return IsDate1(data) || IsDate2(data) || isDate3 || IsDate4(data)
}

func IsDate1(data string) bool {
	return regexpDate1.MatchString(data)
}

func IsDate2(data string) bool {
	return regexpDate2.MatchString(data)
}

func IsDate3(data string) (bool, []string) {
	matchGroups := regexpDate3.FindStringSubmatch(data)
	//return regexpDate3.MatchString(data)
	return len(matchGroups) == 3, matchGroups
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
