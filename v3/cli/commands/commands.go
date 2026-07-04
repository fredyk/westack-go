package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/fredyk/westack-go/v3/cli"
	"github.com/fredyk/westack-go/v3/cli/modelwriter"
	"github.com/fredyk/westack-go/v3/cli/migrator"
)

// supportedTypes lists the valid field type names.
var supportedTypes = map[string]cli.FieldType{
	"string":    cli.TypeString,
	"int":       cli.TypeInt,
	"bool":      cli.TypeBool,
	"time.Time": cli.TypeTime,
	"vector":    cli.TypeVector,
}

// validModelNameRe validates model names: uppercase start, [A-Za-z0-9_], max 128.
var validModelNameRe = regexp.MustCompile(`^[A-Z][A-Za-z0-9_]{0,127}$`)

// ParseField parses a "name:type" string into a cli.Field.
func ParseField(s string) (cli.Field, error) {
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return cli.Field{}, fmt.Errorf("field %q: expected name:type format", s)
	}
	name := s[:idx]
	typeStr := s[idx+1:]

	if strings.TrimSpace(name) == "" {
		return cli.Field{}, fmt.Errorf("field %q: name is empty", s)
	}
	if strings.TrimSpace(typeStr) == "" {
		return cli.Field{}, fmt.Errorf("field %q: type is empty", s)
	}

	ft, ok := supportedTypes[typeStr]
	if !ok {
		if strings.HasPrefix(typeStr, "vector(") && strings.HasSuffix(typeStr, ")") {
			ft = cli.TypeVector
			dimStr := typeStr[7 : len(typeStr)-1]
			dim, err := strconv.Atoi(dimStr)
			if err != nil {
				return cli.Field{}, fmt.Errorf("field %q: invalid vector dimension %q: %w", s, dimStr, err)
			}
			if dim <= 0 {
				return cli.Field{}, fmt.Errorf("field %q: vector dimension must be positive, got %d", s, dim)
			}
			return cli.Field{Name: name, Type: ft, Options: []cli.FieldOption{cli.VectorDim(dim)}}, nil
		}
		return cli.Field{}, fmt.Errorf("field %q: unsupported type %q (supported: string, int, bool, time.Time, vector, vector(N))", s, typeStr)
	}

	return cli.Field{Name: name, Type: ft}, nil
}

// ParseFields parses multiple "name:type" strings.
func ParseFields(inputs []string) ([]cli.Field, error) {
	var fields []cli.Field
	for _, inp := range inputs {
		f, err := ParseField(inp)
		if err != nil {
			return nil, err
		}
		fields = append(fields, f)
	}
	return fields, nil
}

// ParseModelName validates a model name.
func ParseModelName(name string) (string, error) {
	if !validModelNameRe.MatchString(name) {
		return "", fmt.Errorf("model name %q: must start with uppercase letter and contain only [A-Za-z0-9_]", name)
	}
	return name, nil
}

// ModelWriterAlias is a type alias for the modelwriter interface.
type ModelWriterAlias = modelwriter.ModelWriter

// ParseModelNew parses arguments for "model new" command.
func ParseModelNew(modelName string, fieldArgs []string, mw ModelWriterAlias, destDir string) (cli.Model, []cli.Field, error) {
	name, err := ParseModelName(modelName)
	if err != nil {
		return cli.Model{}, nil, err
	}

	fields, err := ParseFields(fieldArgs)
	if err != nil {
		return cli.Model{}, nil, err
	}

	return cli.Model{Name: name, Base: strings.ToLower(name), Fields: fields}, fields, nil
}

// RunModelNew executes the "model new" command: creates the Go model file.
func RunModelNew(model cli.Model, fields []cli.Field, mw ModelWriterAlias, destDir string) error {
	path := destDir + "/" + strings.ToLower(model.Name) + ".wst.go"
	return mw.NewModel(model, path)
}

// RunModelAddField executes the "model add-field" command.
func RunModelAddField(path, modelName string, field cli.Field, mw ModelWriterAlias) error {
	return mw.AddField(path, modelName, field)
}

// RunMigrate executes the "migrate" command: generates DDL.
func RunMigrate(models []cli.Model) string {
	m := migrator.New()
	return m.DDL(models)
}
