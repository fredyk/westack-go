package graphql

import (
	"strconv"
	"strings"
)

// parseFieldArgs localiza el campo de operación `opName` dentro de `query` y
// parsea su lista de argumentos `(nombre: valor, ...)` a un map[string]any.
// Soporta valores escalares (string, int, float, bool, null), enums
// (identificadores), objetos `{...}` y listas `[...]` de forma recursiva.
// Devuelve nil si la operación no tiene argumentos.
func parseFieldArgs(query, opName string) map[string]any {
	// Guard: opName vacío haría que strings.Index(q, "") devuelva 0 en cada
	// iteración con `from` sin avanzar → bucle infinito (DoS). Sin nombre de
	// operación no hay args que parsear.
	if opName == "" {
		return nil
	}
	// Buscar `opName` seguido (tras espacios) de '('. Recorremos todas las
	// apariciones por si el nombre aparece antes en un alias/comentario.
	from := 0
	for {
		i := strings.Index(query[from:], opName)
		if i < 0 {
			return nil
		}
		i += from
		j := i + len(opName)
		// El carácter anterior no debe ser parte de un identificador mayor.
		if i > 0 && isIdentChar(query[i-1]) {
			from = j
			continue
		}
		// Saltar espacios hasta el siguiente carácter significativo.
		k := j
		for k < len(query) && isSpace(query[k]) {
			k++
		}
		if k < len(query) && query[k] == '(' {
			p := &argParser{s: query, pos: k + 1}
			return p.parseArgList()
		}
		// No lleva args (p. ej. `expediente { ... }`).
		if k < len(query) && (query[k] == '{' || query[k] == '}') {
			return nil
		}
		from = j
	}
}

type argParser struct {
	s   string
	pos int
}

// parseArgList parsea `nombre: valor, nombre: valor ...)` hasta el ')'.
func (p *argParser) parseArgList() map[string]any {
	args := map[string]any{}
	for {
		p.skipSpaceAndCommas()
		if p.pos >= len(p.s) || p.s[p.pos] == ')' {
			p.pos++ // consumir ')'
			break
		}
		name := p.parseName()
		if name == "" {
			break
		}
		p.skipSpace()
		if p.pos < len(p.s) && p.s[p.pos] == ':' {
			p.pos++
		}
		p.skipSpace()
		args[name] = p.parseValue()
	}
	if len(args) == 0 {
		return nil
	}
	return args
}

func (p *argParser) parseValue() any {
	p.skipSpace()
	if p.pos >= len(p.s) {
		return nil
	}
	switch c := p.s[p.pos]; {
	case c == '"':
		return p.parseString()
	case c == '{':
		return p.parseObject()
	case c == '[':
		return p.parseList()
	case c == '$':
		// Variable de GraphQL: no resuelta aquí (se resuelve fuera con Variables).
		p.pos++
		return "$" + p.parseName()
	default:
		return p.parseScalarOrEnum()
	}
}

func (p *argParser) parseString() string {
	p.pos++ // abrir comilla
	var b strings.Builder
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '\\' && p.pos+1 < len(p.s) {
			p.pos++
			esc := p.s[p.pos]
			switch esc {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '"', '\\', '/':
				b.WriteByte(esc)
			default:
				b.WriteByte(esc)
			}
			p.pos++
			continue
		}
		if c == '"' {
			p.pos++ // cerrar comilla
			break
		}
		b.WriteByte(c)
		p.pos++
	}
	return b.String()
}

func (p *argParser) parseObject() map[string]any {
	p.pos++ // '{'
	obj := map[string]any{}
	for {
		p.skipSpaceAndCommas()
		if p.pos >= len(p.s) || p.s[p.pos] == '}' {
			p.pos++
			break
		}
		name := p.parseName()
		p.skipSpace()
		if p.pos < len(p.s) && p.s[p.pos] == ':' {
			p.pos++
		}
		obj[name] = p.parseValue()
	}
	return obj
}

func (p *argParser) parseList() []any {
	p.pos++ // '['
	var list []any
	for {
		p.skipSpaceAndCommas()
		if p.pos >= len(p.s) || p.s[p.pos] == ']' {
			p.pos++
			break
		}
		list = append(list, p.parseValue())
	}
	return list
}

// parseScalarOrEnum lee un token hasta un delimitador y lo interpreta como
// número, booleano, null o enum/identificador (string).
func (p *argParser) parseScalarOrEnum() any {
	start := p.pos
	for p.pos < len(p.s) && !isDelim(p.s[p.pos]) {
		p.pos++
	}
	tok := strings.TrimSpace(p.s[start:p.pos])
	switch tok {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if n, err := strconv.ParseInt(tok, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(tok, 64); err == nil {
		return f
	}
	return tok // enum / identificador
}

func (p *argParser) parseName() string {
	p.skipSpace()
	start := p.pos
	for p.pos < len(p.s) && isIdentChar(p.s[p.pos]) {
		p.pos++
	}
	return p.s[start:p.pos]
}

func (p *argParser) skipSpace() {
	for p.pos < len(p.s) && isSpace(p.s[p.pos]) {
		p.pos++
	}
}

func (p *argParser) skipSpaceAndCommas() {
	for p.pos < len(p.s) && (isSpace(p.s[p.pos]) || p.s[p.pos] == ',') {
		p.pos++
	}
}

func isSpace(c byte) bool     { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }
func isIdentChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
func isDelim(c byte) bool {
	return isSpace(c) || c == ',' || c == ')' || c == '}' || c == ']' || c == ':' || c == '{' || c == '('
}
