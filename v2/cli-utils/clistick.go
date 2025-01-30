package cliutils

import (
	"log"
	"strings"

	"github.com/fredyk/westack-go/v2/model"
	"github.com/tyler-sommer/stick"
)

func addStickFunctions(config model.Config, configs []model.Config, stickEnv *stick.Env, data map[string]stick.Value) {
	stickEnv.Functions["capitalize"] = func(ctx stick.Context, args ...stick.Value) stick.Value {
		s := args[0].(string)
		return strings.ToUpper(s[:1]) + s[1:]
	}

	stickEnv.Functions["renderImports"] = func(ctx stick.Context, args ...stick.Value) stick.Value {
		includeStructImports := args[0].(bool)
		neededImports := processNeededImports(config, includeStructImports)
		additionalImports := make([]string, 0)
		for idx, c := range args {
			if idx >= 1 {
				additionalImports = append(additionalImports, "\""+c.(string)+"\"")
			}
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
