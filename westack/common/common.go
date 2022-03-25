package wst

import (
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"io/ioutil"
	"regexp"
	"strings"
)

type M map[string]interface{}

type A []M

type Where M

type Include []struct {
	Relation string  `json:"relation"`
	Scope    *Filter `json:"scope"`
}

type Order []string

type Filter struct {
	Where   *Where   `json:"where"`
	Include *Include `json:"include"`
	Order   *Order   `json:"order"`
	Skip    int64    `json:"skip"`
	Limit   int64    `json:"limit"`
}

type IApp struct {
	Debug        bool
	SwaggerPaths func() *map[string]M
	FindModel    func(modelName string) interface{}
	JwtSecretKey []byte
}

func LoadFile(filePath string, out interface{}) error {
	jsonFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	//var result M
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
	// Create the target map
	targetMap := make(M)

	// Copy from the original map to the target map
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
