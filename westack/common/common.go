package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
)

type IApp struct {
	SwaggerPaths func() *map[string]map[string]interface{}
}

func LoadFile(filePath string, out interface{}) interface{} {
	jsonFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
	}

	//var result map[string]interface{}
	err2 := json.Unmarshal(jsonFile, &out)
	if err2 != nil {
		fmt.Println(err2)
	}
	return out
}

func DashedCase(st string) string {
	var res = strings.ToLower(st[:1])
	compile, err := regexp.Compile("([A-Z])")
	if err != nil {
		log.Println(err)
		return ""
	}
	res += string(compile.ReplaceAllFunc([]byte(st[1:]), func(bytes []byte) []byte {
		return []byte("-" + strings.ToLower(string(bytes[0])))
	}))
	return res
}

func CopyMap(src map[string]interface{}) map[string]interface{} {
	// Create the target map
	targetMap := make(map[string]interface{})

	// Copy from the original map to the target map
	for key, value := range src {
		targetMap[key] = value
	}
	return targetMap
}
