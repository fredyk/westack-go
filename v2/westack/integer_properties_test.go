package westack

import (
	"encoding/json"
	"math"
	"testing"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/stretchr/testify/assert"
)

func integerTestProperties() map[string]model.Property {
	return map[string]model.Property{
		"asInt":     {Type: "int"},
		"asInteger": {Type: "integer"},
		"asInt64":   {Type: "int64"},
		"asNumber":  {Type: "number"},
		"asString":  {Type: "string"},
	}
}

func Test_coerceIntegerProperties_WholeFloat64BecomesInt64(t *testing.T) {
	data := wst.M{"asInt": float64(3), "asInteger": float64(1002000), "asInt64": float64(-7)}

	codes := coerceIntegerProperties(&data, integerTestProperties())

	assert.Empty(t, codes)
	assert.Equal(t, int64(3), data["asInt"])
	assert.Equal(t, int64(1002000), data["asInteger"])
	assert.Equal(t, int64(-7), data["asInt64"])
}

func Test_coerceIntegerProperties_JsonNumberAndGoIntsBecomeInt64(t *testing.T) {
	data := wst.M{
		"asInt":     json.Number("42"),
		"asInteger": 5,
		"asInt64":   json.Number("1.002e+06"),
	}

	codes := coerceIntegerProperties(&data, integerTestProperties())

	assert.Empty(t, codes)
	assert.Equal(t, int64(42), data["asInt"])
	assert.Equal(t, int64(5), data["asInteger"])
	assert.Equal(t, int64(1002000), data["asInt64"])
}

func Test_coerceIntegerProperties_OtherIntegerKindsBecomeInt64(t *testing.T) {
	for _, value := range []any{int8(9), int16(9), int32(9), int64(9), uint(9), uint8(9), uint16(9), uint32(9), uint64(9), float32(9)} {
		data := wst.M{"asInteger": value}

		codes := coerceIntegerProperties(&data, integerTestProperties())

		assert.Empty(t, codes, "%T", value)
		assert.Equal(t, int64(9), data["asInteger"], "%T", value)
	}
}

func Test_coerceIntegerProperties_FractionalValueIsRejectedPerField(t *testing.T) {
	data := wst.M{"asInt": 1.5, "asInteger": json.Number("2.25"), "asInt64": float64(4)}

	codes := coerceIntegerProperties(&data, integerTestProperties())

	assert.Equal(t, wst.M{"asInt": []string{"integer"}, "asInteger": []string{"integer"}}, codes)
}

func Test_coerceIntegerProperties_NonFiniteAndOutOfRangeAreRejected(t *testing.T) {
	for _, value := range []any{math.NaN(), math.Inf(1), math.Inf(-1), 1e300, uint64(math.MaxUint64)} {
		data := wst.M{"asInt64": value}

		codes := coerceIntegerProperties(&data, integerTestProperties())

		assert.Equal(t, wst.M{"asInt64": []string{"integer"}}, codes, "%v", value)
	}
}

func Test_coerceIntegerProperties_NilAbsentAndOtherTypesAreUntouched(t *testing.T) {
	data := wst.M{"asInt": nil, "asNumber": 1.5, "asString": float64(3)}

	codes := coerceIntegerProperties(&data, integerTestProperties())

	assert.Empty(t, codes)
	assert.Nil(t, data["asInt"])
	_, present := data["asInteger"]
	assert.False(t, present)
	assert.Equal(t, 1.5, data["asNumber"])
	assert.Equal(t, float64(3), data["asString"])
}
