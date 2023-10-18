package wst

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mailru/easyjson/jlexer"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var NilBytes = []byte{'n', 'u', 'l', 'l'}

type M map[string]interface{}

func (m *M) GetM(key string) *M {
	if m == nil {
		return nil
	}
	if v, ok := (*m)[key]; ok {
		if vv, ok := v.(M); ok {
			return &vv
		} else if vv, ok := v.(map[string]interface{}); ok {
			var out = make(M, len(vv))
			for k, v := range vv {
				out[k] = v
			}
			return &out
		}
	}
	return nil
}

func (m *M) GetString(path string) string {
	if m == nil {
		return ""
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		v := (*m)[segments[0]]
		if v == nil {
			return ""
		}
		if vv, ok := v.(string); ok {
			return vv
		} else if vv, ok := v.(primitive.ObjectID); ok {
			return vv.Hex()
		}
	} else {
		source := obtainSourceFromM(m, segments[:len(segments)-1])
		if v, ok := source.(M); ok {
			return v.GetString(segments[len(segments)-1])
		} else if v, ok := source.(map[string]interface{}); ok {
			vv := v[segments[len(segments)-1]]
			return vv.(string)
		}
	}
	return ""
}

func (m *M) GetInt(path string) int {
	if m == nil {
		return 0
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return asInt((*m)[segments[0]])
	} else {
		source := obtainSourceFromM(m, segments[:len(segments)-1])
		if v, ok := source.(M); ok {
			return v.GetInt(segments[len(segments)-1])
		} else if v, ok := source.(map[string]interface{}); ok {
			return asInt(v[segments[len(segments)-1]])
		}
	}
	return 0
}

func (m *M) MarshalEasyJSON(w *jwriter.Writer) {
	if m == nil {
		w.Raw(NilBytes, nil)
		return
	}
	w.RawByte('{')
	first := true
	for k, v := range *m {
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
		case primitive.ObjectID:
			w.String(v.Hex())
		case primitive.DateTime:
			// format in time.RFC3339Nano v.(primitive.DateTime).Time()
			w.String(v.Time().Format(time.RFC3339Nano))
		default:
			if vv, ok := v.(easyjson.Marshaler); ok {
				//fmt.Printf("Found easyjson.Marshaler: %v at %v\n", vv, k)
				vv.MarshalEasyJSON(w)
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
	l.Delim('{')
	*m = make(M)
	for {
		l.FetchToken()
		if l.IsDelim('}') {
			return
		}
		key := l.String()
		l.WantColon()
		l.FetchToken()
		rawValue := l.Raw()
		if rawValue[0] == '{' {
			subM := make(M)
			//goland:noinspection GoUnhandledErrorResult
			easyjson.Unmarshal(rawValue, &subM) // #nosec G601
			//if err != nil {
			//	l.AddError(err)
			//	return
			//}
			(*m)[key] = subM
		} else if rawValue[0] == '[' {
			subA := make(A, 0)
			//goland:noinspection GoUnhandledErrorResult
			easyjson.Unmarshal(rawValue, &subA) // #nosec G601
			//if err != nil {
			//	l.AddError(err)
			//	return
			//}
			(*m)[key] = subA
		} else {
			var v interface{}
			//goland:noinspection GoUnhandledErrorResult
			json.Unmarshal(rawValue, &v)
			//if err != nil {
			//	l.AddError(err)
			//	return
			//}
			(*m)[key] = v
		}
		l.WantComma()
		//if l.Error() != nil {
		//	return
		//}
	}
}

func (m *M) GetBoolean(path string) bool {
	if m == nil {
		return false
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return (*m)[segments[0]].(bool)
	} else {
		source := obtainSourceFromM(m, segments[:len(segments)-1])
		if v, ok := source.(M); ok {
			return v[segments[len(segments)-1]].(bool)
		}
	}
	return false
}

func asInt(v interface{}) int {
	switch v := v.(type) {
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

type A []M

//func (a *A) MarshalEasyJSON(w *jwriter.Writer) {
//	if a == nil {
//		w.Raw(NilBytes, nil)
//		return
//	}
//	w.RawByte('[')
//	first := true
//	for _, v := range *a {
//		if first {
//			first = false
//		} else {
//			w.RawByte(',')
//		}
//		bytes, err := easyjson.Marshal(&v) // #nosec G601
//		w.Raw(bytes, err)
//	}
//	w.RawByte(']')
//}

func (a *A) UnmarshalEasyJSON(l *jlexer.Lexer) {
	if l.IsNull() {
		l.Skip()
		return
	}
	l.Delim('[')
	*a = make(A, 0)
	for {
		l.FetchToken()
		if l.IsDelim(']') {
			return
		}
		rawValue := l.Raw()
		if rawValue[0] == '{' {
			subM := make(M)
			//goland:noinspection GoUnhandledErrorResult
			easyjson.Unmarshal(rawValue, &subM) // #nosec G601
			//if err != nil {
			//	l.AddError(err)
			//	return
			//}
			*a = append(*a, subM)
		} else {
			var v interface{}
			//goland:noinspection GoUnhandledErrorResult
			json.Unmarshal(rawValue, &v)
			//if err != nil {
			//	l.AddError(err)
			//	return
			//}
			*a = append(*a, M{
				"<value>": v,
			})
		}
		l.WantComma()
		//if l.Error() != nil {
		//	return
		//}
	}
}

func (a *A) GetAt(idx int) *M {
	if a == nil {
		return nil
	}
	if idx < 0 || idx >= len(*a) {
		return nil
	}
	return &(*a)[idx]
}

func (a *A) GetM(path string) *M {
	if a == nil {
		return nil
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return a.GetAt(obtainIdxFromSegment(segments[0]))
	} else {
		source := obtainSourceFromA(a, segments[:len(segments)-1])
		if v, ok := source.(M); ok {
			vv := v[segments[len(segments)-1]].(M)
			return &vv
		} else if v, ok := source.(A); ok {
			return v.GetAt(obtainIdxFromSegment(segments[len(segments)-1]))
		}
	}
	return nil
}

func (a *A) GetString(path string) string {
	if a == nil {
		return ""
	}
	segments := strings.Split(path, ".")
	segmentIdx := 0
	var source interface{}
	for {
		if source == nil {
			if segmentIdx == 0 {
				source = obtainSourceFromA(a, segments[:1])
			} else {
				return ""
			}
		} else if v, ok := source.(A); ok {
			source = obtainSourceFromA(&v, segments[segmentIdx:segmentIdx+1])
		} else if v, ok := source.(M); ok {
			return v.GetString(strings.Join(segments[segmentIdx:], "."))
		} else {
			log.Printf("WARNING: GetString: not an M: %v\n", reflect.TypeOf(source))
			return ""
		}
		segmentIdx++
	}
}

func obtainSourceFromA(a *A, segments []string) interface{} {
	if len(segments) > 1 {
		var prevSource = obtainSourceFromA(a, segments[:len(segments)-1])
		if v, ok := prevSource.(M); ok {
			return v[segments[len(segments)-1]]
		} else if v, ok := prevSource.(A); ok {
			// last segment is an index wrapped by brackets
			// e.g. "a.b[0].c"
			// so we need to remove the brackets
			idx := obtainIdxFromSegment(segments[len(segments)-1])
			if idx < 0 || idx >= len(v) {
				return nil
			}
			return v[idx]
		}
	}
	return (*a)[obtainIdxFromSegment(segments[0])]
}

func obtainSourceFromM(m *M, segments []string) interface{} {
	if len(segments) > 1 {
		var prevSource = obtainSourceFromM(m, segments[:len(segments)-1])
		if v, ok := prevSource.(M); ok {
			return v[segments[len(segments)-1]]
		} else if v, ok := prevSource.(A); ok {
			// last segment is an index wrapped by brackets
			// e.g. "a.b[0].c"
			// so we need to remove the brackets
			idx := obtainIdxFromSegment(segments[len(segments)-1])
			if idx < 0 || idx >= len(v) {
				return nil
			}
			return v[idx]
		}
	}
	return (*m)[segments[0]]
}

func obtainIdxFromSegment(segment string) int {
	segment = segment[1 : len(segment)-1]
	idx, _ := strconv.Atoi(segment)
	return idx
}

func GetTypedList[T any](m *M, path string) []T {
	if m == nil {
		return nil
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return (*m)[segments[0]].([]T)
	} else {
		source := obtainSourceFromM(m, segments[:len(segments)-1])
		if v, ok := source.(M); ok {
			return v[segments[len(segments)-1]].([]T)
		} else {
			log.Printf("WARNING: GetTypedList: not an M: %v\n", reflect.TypeOf(source))
			return nil
		}
	}
}

func GetTypedItem[T any](m *M, path string) T {
	var defaultResult T
	if m == nil {
		return defaultResult
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return (*m)[segments[0]].(T)
	} else {
		source := obtainSourceFromM(m, segments[:len(segments)-1])
		if v, ok := source.(M); ok {
			return v[segments[len(segments)-1]].(T)
		} else {
			log.Printf("WARNING: GetTypedItem: not an M: %v\n", reflect.TypeOf(source))
			return defaultResult
		}
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
	NilMap          = M{"<wst.NilMap>": 1}
	DashedCaseRegex = regexp.MustCompile("([A-Z])")
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

// AFromPrimitiveSlice converts a primitive slice of primitive.M or M to a *A
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
	SwaggerHelper  func() SwaggerHelper
	FindModel      func(modelName string) (interface{}, error)
	FindDatasource func(datasource string) (interface{}, error)
	Logger         func() ILogger
	JwtSecretKey   []byte
	Viper          *viper.Viper
	Bson           BsonOptions
}

var RegexpIdEntire = regexp.MustCompile(`^([0-9a-f]{24})$`)
var RegexpIpStart = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`)

var regexpTrimMilliseconds = regexp.MustCompile(`^(.+\.\d{3})\d+(.*)$`)
var regexpTimeZoneReplacing = regexp.MustCompile(`([+\-]\d{2}):(\d{2})$`)
var regexpDate1 = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})([+\-:0-9]+)$`)
var regexpDate2 = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3,})([+\-:0-9]+)$`)
var regexpDate3 = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(Z)?$`)
var regexpDate4 = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3,})(Z)?$`)

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

// Transform High-cost operation...
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
	prevData := data
	data = regexpTrimMilliseconds.ReplaceAllString(data, "$1$2")
	if prevData != data {
		log.Printf("WARNING: ParseDate: trimming input to 3 digits of milliseconds: %v -> %v", prevData, data)
	}
	if IsDate1(data) {
		layout := "2006-01-02T15:04:05-0700"
		parsedDate, err = time.Parse(layout, regexpTimeZoneReplacing.ReplaceAllString(data, "$1$2"))
	} else if IsDate2(data) {
		layout := "2006-01-02T15:04:05.000-0700"
		// Trim input up to 3 digits of milliseconds using regex
		// This is needed because time.Parse() does not support more than 3 digits of milliseconds

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

type ILogger interface {
	Printf(format string, v ...any)
	Print(v ...any)
	Println(v ...any)
	Fatal(v ...any)
	Fatalf(format string, v ...any)
	Fatalln(v ...any)
	Panic(v ...any)
	Panicf(format string, v ...any)
	Panicln(v ...any)
	Flags() int
	SetFlags(flag int)
	Prefix() string
	SetPrefix(prefix string)
}

type objectIdCodec struct{}

func (objectIdCodec) EncodeValue(_ bsoncodec.EncodeContext, writer bsonrw.ValueWriter, val reflect.Value) error {
	oid := val.Interface().(primitive.ObjectID)
	return writer.WriteObjectID(oid)
}

func (objectIdCodec) DecodeValue(_ bsoncodec.DecodeContext, reader bsonrw.ValueReader, val reflect.Value) error {
	var oid primitive.ObjectID
	var err error
	switch reader.Type() {
	case bson.TypeObjectID:
		oid, err = reader.ReadObjectID()
		if err != nil {
			return err
		}
	//case bson.TypeString:
	//	str, err := reader.ReadString()
	//	if err != nil {
	//		return err
	//	}
	//	oid, err = primitive.ObjectIDFromHex(str)
	//	if err != nil {
	//		return err
	//	}
	default:
		return fmt.Errorf("cannot decode %v into a primitive.ObjectID", reader.Type())
	}
	val.Set(reflect.ValueOf(oid))
	return nil
}

func newObjectIDCodec() bsoncodec.ValueCodec {
	return &objectIdCodec{}
}

func CreateDefaultMongoRegistry() *bsoncodec.Registry {
	// create a new registry
	bsonRegistryBuilder := bson.NewRegistryBuilder().
		RegisterCodec(reflect.TypeOf(primitive.ObjectID{}), newObjectIDCodec()). // register the primitive.ObjectID type
		//RegisterTypeMapEntry(bson.TypeEmbeddedDocument, reflect.TypeOf(bson.M{})).
		RegisterTypeMapEntry(bson.TypeEmbeddedDocument, reflect.TypeOf(M{})).
		//RegisterTypeMapEntry(bson.TypeArray, reflect.TypeOf([]bson.M{}))
		RegisterTypeMapEntry(bson.TypeArray, reflect.TypeOf(A{}))

	// register the custom types
	bsonRegistryBuilder.
		RegisterTypeEncoder(reflect.TypeOf(time.Time{}), bsoncodec.ValueEncoderFunc(func(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
			return vw.WriteDateTime(val.Interface().(time.Time).UnixNano() / int64(time.Millisecond))
		})).
		RegisterTypeDecoder(reflect.TypeOf(time.Time{}), bsoncodec.ValueDecoderFunc(func(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
			var unixNano int64
			var err error
			switch vr.Type() {
			case bson.TypeDateTime:
				unixNano, err = vr.ReadDateTime()
				if err != nil {
					return err
				}
			//case bson.TypeInt64:
			//	var int64Val int64
			//	int64Val, err = vr.ReadInt64()
			//	if err != nil {
			//		return err
			//	}
			//	unixNano = int64Val
			default:
				return fmt.Errorf("cannot decode %v into a time.Time", vr.Type())
			}
			val.Set(reflect.ValueOf(time.Unix(0, unixNano*int64(time.Millisecond))))
			return nil
		}))

	return bsonRegistryBuilder.Build()
}
