package swaggerhelper

import (
	"encoding/json"
	"reflect"
	"testing"

	wst "github.com/fredyk/westack-go/v2/common"
)

type sampleWithByteSlices struct {
	Raw     json.RawMessage `json:"raw"`
	Bytes   []byte          `json:"bytes"`
	Small   []uint16        `json:"small"`
	Signed  []int8          `json:"signed"`
	Ratios  []float32       `json:"ratios"`
	Counter uint64          `json:"counter"`
}

// encoding/json writes a []byte as a base64 string and a json.RawMessage verbatim, as any JSON
// value; neither is an array of numbers, and describing them must not panic.
func TestByteSlicesAndUnsignedIntegersAreDescribed(t *testing.T) {
	components := wst.M{"schemas": wst.M{}}
	var schema wst.M
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("analyzeWithReflection panicked: %v", r)
			}
		}()
		schema = analyzeWithReflection("", reflect.TypeOf(sampleWithByteSlices{}), &components)
	}()
	want := map[string]wst.M{
		"raw":     {},
		"bytes":   {"type": "string", "format": "byte"},
		"small":   {"type": "array", "items": wst.M{"type": "number"}},
		"signed":  {"type": "array", "items": wst.M{"type": "number"}},
		"ratios":  {"type": "array", "items": wst.M{"type": "number"}},
		"counter": {"type": "number"},
	}
	for field, w := range want {
		if got, _ := schema[field].(wst.M); !reflect.DeepEqual(got, w) {
			t.Errorf("%s: got %v, want %v", field, schema[field], w)
		}
	}
}
