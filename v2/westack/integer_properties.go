package westack

import (
	"encoding/json"
	"math"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
)

// integerValidationCode is the per-field code reported when a property
// declared as an integer receives a value that is not a whole number.
const integerValidationCode = "integer"

func isIntegerPropertyType(propertyType interface{}) bool {
	switch propertyType {
	case "int", "integer", "int64":
		return true
	}
	return false
}

// coerceIntegerProperties stores every value of a property declared as
// "int", "integer" or "int64" as an int64, so that the datasource persists it
// as an integer instead of a double (JSON bodies decode numbers as float64).
// Nil and absent values are left untouched. A value that is not a whole number
// that fits in an int64 is reported under its property name with the
// "integer" code, and left as it was.
func coerceIntegerProperties(data *wst.M, properties map[string]model.Property) wst.M {
	codes := wst.M{}
	if data == nil {
		return codes
	}
	for propertyName, propertyConfig := range properties {
		if !isIntegerPropertyType(propertyConfig.Type) {
			continue
		}
		value, present := (*data)[propertyName]
		if !present || value == nil {
			continue
		}
		converted, ok, handled := toInt64(value)
		if !handled {
			continue
		}
		if !ok {
			codes[propertyName] = []string{integerValidationCode}
			continue
		}
		(*data)[propertyName] = converted
	}
	return codes
}

// toInt64 converts a numeric value to int64. handled is false for values that
// are not numbers; ok is false for numbers that are not whole or do not fit.
func toInt64(value interface{}) (converted int64, ok bool, handled bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true, true
	case int8:
		return int64(v), true, true
	case int16:
		return int64(v), true, true
	case int32:
		return int64(v), true, true
	case int64:
		return v, true, true
	case uint:
		return uint64ToInt64(uint64(v))
	case uint8:
		return int64(v), true, true
	case uint16:
		return int64(v), true, true
	case uint32:
		return int64(v), true, true
	case uint64:
		return uint64ToInt64(v)
	case float32:
		return float64ToInt64(float64(v))
	case float64:
		return float64ToInt64(v)
	case json.Number:
		if asInt, err := v.Int64(); err == nil {
			return asInt, true, true
		}
		asFloat, err := v.Float64()
		if err != nil {
			return 0, false, true
		}
		return float64ToInt64(asFloat)
	}
	return 0, false, false
}

func uint64ToInt64(v uint64) (int64, bool, bool) {
	if v > math.MaxInt64 {
		return 0, false, true
	}
	return int64(v), true, true
}

func float64ToInt64(v float64) (int64, bool, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v != math.Trunc(v) {
		return 0, false, true
	}
	// 2^63 is exactly representable as float64; anything at or beyond it overflows.
	if v >= math.Pow(2, 63) || v < -math.Pow(2, 63) {
		return 0, false, true
	}
	return int64(v), true, true
}
