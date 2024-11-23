package client

import (
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/goccy/go-json"
	"github.com/mailru/easyjson"
	"reflect"
)

func convertToMap(src interface{}) (wst.M, error) {
	out := wst.M{}

	if src == nil || reflect.ValueOf(src).IsNil() {
		return out, nil
	}
	b, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	err = easyjson.Unmarshal(b, &out)
	return out, err
}
