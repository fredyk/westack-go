package cliutils

import (
	"github.com/fredyk/westack-go/v2/model"
	"github.com/tyler-sommer/stick"
	"log"
	"strings"
)

func addStickFunctions(config model.Config, configs []model.Config, stickEnv *stick.Env, data map[string]stick.Value) {
	stickEnv.Functions["capitalize"] = func(ctx stick.Context, args ...stick.Value) stick.Value {
		s := args[0].(string)
		return strings.ToUpper(s[:1]) + s[1:]
	}

	stickEnv.Functions["renderImports"] = func(ctx stick.Context, args ...stick.Value) stick.Value {
		neededImports := processNeededImports(config)
		additionalImports := make([]string, 0)
		for _, c := range args {
			additionalImports = append(additionalImports, "\""+c.(string)+"\"")
		}
		neededImports = append(neededImports, additionalImports...)
		if len(neededImports) == 0 {
			return "import _ \"embed\"\n"
		} else {
			return "import (\n\t_ \"embed\"\n\t" + strings.Join(neededImports, "\n\t") + "\n)\n"
		}
	}

	stickEnv.Functions["renderStruct"] = func(ctx stick.Context, args ...stick.Value) stick.Value {
		var outWriter strings.Builder
		err := stickEnv.Execute(StructTemplate, &outWriter, data)
		if err != nil {
			log.Println(err)
		}
		return outWriter.String()
	}

	stickEnv.Functions["renderType"] = func(ctx stick.Context, args ...stick.Value) stick.Value {
		s := args[0].(string)
		switch s {
		case "number", "integer":
			return "int"
		case "float":
			return "float64"
		case "date":
			return "time.Time"
		case "boolean":
			return "bool"
		case "list":
			return "[]string"
		case "map":
			return "map[string]string"
		default:
			return "string"
		}
	}
}
