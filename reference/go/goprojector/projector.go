// Package goprojector renders a bounded canonical Seme program directly as
// ordinary Go source. It deliberately rejects semantics outside its declared
// profile instead of guessing a Go spelling.
package goprojector

import (
	"encoding/hex"
	"fmt"
	"go/format"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	sFunction      = "00000000000000000000000000009011"
	sParameter     = "00000000000000000000000000009012"
	sRead          = "00000000000000000000000000009013"
	sAdd           = "00000000000000000000000000009014"
	sProgram       = "00000000000000000000000000009015"
	sInteger       = "00000000000000000000000000009010"
	sBoolean       = "00000000000000000000000000009020"
	sString        = "00000000000000000000000000009040"
	sStringLiteral = "00000000000000000000000000009050"
	sConcat        = "000000000000000000000000000090c3"
	sBlock         = "00000000000000000000000000009080"
	sReturn        = "00000000000000000000000000009081"
	sCall          = "00000000000000000000000000009060"
)

type entity struct {
	id, schema string
	fields     map[string][]string
}
type context struct {
	graph      map[string]entity
	functions  map[string]string
	parameters map[string]string
}

// Project emits one gofmt-formatted file. packageName is projection metadata,
// not canonical meaning, and must be a valid Go identifier.
func Project(g1 []byte, packageName string) ([]byte, error) {
	if !identifier(packageName) {
		return nil, fmt.Errorf("go_projection.invalid_package_name")
	}
	graph, err := parse(g1)
	if err != nil {
		return nil, err
	}
	var programs []entity
	for _, e := range graph {
		if e.schema == sProgram {
			programs = append(programs, e)
		}
	}
	if len(programs) != 1 {
		return nil, fmt.Errorf("go_projection.requires_one_program")
	}
	ids, err := refs(programs[0], "00000000000000000000000000009150")
	if err != nil || len(ids) == 0 {
		return nil, fmt.Errorf("go_projection.invalid_program_members")
	}
	entry, err := ref(programs[0], "00000000000000000000000000009151")
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	seen := map[string]bool{}
	for _, id := range ids {
		fn, ok := graph[id]
		if !ok || fn.schema != sFunction {
			return nil, fmt.Errorf("go_projection.invalid_function_member")
		}
		name, err := text(fn, "00000000000000000000000000009110")
		if err != nil || !identifier(name) || seen[name] {
			return nil, fmt.Errorf("go_projection.invalid_function_name")
		}
		names[id], seen[name] = name, true
	}
	if _, ok := names[entry]; !ok {
		return nil, fmt.Errorf("go_projection.entry_membership")
	}
	var out strings.Builder
	fmt.Fprintf(&out, "package %s\n\n", packageName)
	for _, id := range ids {
		fn := graph[id]
		paramIDs, err := refs(fn, "00000000000000000000000000009111")
		if err != nil {
			return nil, err
		}
		params := map[string]string{}
		var declarations []string
		for _, pid := range paramIDs {
			p, ok := graph[pid]
			if !ok || p.schema != sParameter {
				return nil, fmt.Errorf("go_projection.invalid_parameter")
			}
			name, e := text(p, "00000000000000000000000000009120")
			if e != nil || !identifier(name) {
				return nil, fmt.Errorf("go_projection.invalid_parameter_name")
			}
			tid, e := ref(p, "00000000000000000000000000009121")
			if e != nil {
				return nil, e
			}
			typ, e := typeName(graph, tid)
			if e != nil {
				return nil, e
			}
			params[pid] = name
			declarations = append(declarations, name+" "+typ)
		}
		resultID, e := ref(fn, "00000000000000000000000000009112")
		if e != nil {
			return nil, e
		}
		result, e := typeName(graph, resultID)
		if e != nil {
			return nil, e
		}
		bodyID, e := ref(fn, "00000000000000000000000000009113")
		if e != nil {
			return nil, e
		}
		body, e := projectBlock(bodyID, context{graph, names, params})
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&out, "func %s(%s) %s {\n%s\n}\n\n", names[id], strings.Join(declarations, ", "), result, body)
	}
	formatted, err := format.Source([]byte(out.String()))
	if err != nil {
		return nil, fmt.Errorf("go_projection.format: %w", err)
	}
	return formatted, nil
}

func projectBlock(id string, c context) (string, error) {
	b, ok := c.graph[id]
	if !ok || b.schema != sBlock {
		return "", fmt.Errorf("go_projection.invalid_block")
	}
	statements, err := refs(b, "00000000000000000000000000009800")
	if err != nil || len(statements) != 1 {
		return "", fmt.Errorf("go_projection.unsupported_block")
	}
	r, ok := c.graph[statements[0]]
	if !ok || r.schema != sReturn {
		return "", fmt.Errorf("go_projection.unsupported_statement")
	}
	values, err := refs(r, "00000000000000000000000000009810")
	if err != nil || len(values) != 1 {
		return "", fmt.Errorf("go_projection.return_arity")
	}
	expression, err := expr(values[0], c)
	if err != nil {
		return "", err
	}
	return "\treturn " + expression, nil
}

func expr(id string, c context) (string, error) {
	e, ok := c.graph[id]
	if !ok {
		return "", fmt.Errorf("go_projection.missing_expression")
	}
	switch e.schema {
	case sRead:
		pid, err := ref(e, "00000000000000000000000000009130")
		if err != nil {
			return "", err
		}
		name, ok := c.parameters[pid]
		if !ok {
			return "", fmt.Errorf("go_projection.parameter_scope")
		}
		return name, nil
	case sCall:
		fid, err := ref(e, "00000000000000000000000000009600")
		if err != nil {
			return "", err
		}
		name, ok := c.functions[fid]
		if !ok {
			return "", fmt.Errorf("go_projection.call_membership")
		}
		args, err := refs(e, "00000000000000000000000000009601")
		if err != nil {
			return "", err
		}
		rendered := make([]string, len(args))
		for i, arg := range args {
			rendered[i], err = expr(arg, c)
			if err != nil {
				return "", err
			}
		}
		return name + "(" + strings.Join(rendered, ", ") + ")", nil
	case sStringLiteral:
		value, err := text(e, "00000000000000000000000000009500")
		if err != nil {
			return "", err
		}
		return strconv.Quote(value), nil
	case sAdd, sConcat:
		leftField, rightField := "00000000000000000000000000009140", "00000000000000000000000000009141"
		if e.schema == sConcat {
			leftField, rightField = "00000000000000000000000000009c30", "00000000000000000000000000009c31"
		}
		left, err := ref(e, leftField)
		if err != nil {
			return "", err
		}
		right, err := ref(e, rightField)
		if err != nil {
			return "", err
		}
		l, err := expr(left, c)
		if err != nil {
			return "", err
		}
		r, err := expr(right, c)
		if err != nil {
			return "", err
		}
		return "(" + l + " + " + r + ")", nil
	default:
		return "", fmt.Errorf("go_projection.unsupported_expression")
	}
}

func typeName(g map[string]entity, id string) (string, error) {
	e, ok := g[id]
	if !ok {
		return "", fmt.Errorf("go_projection.missing_type")
	}
	switch e.schema {
	case sInteger:
		return "int64", nil
	case sBoolean:
		return "bool", nil
	case sString:
		return "string", nil
	}
	return "", fmt.Errorf("go_projection.unsupported_type")
}
func identifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func parse(source []byte) (map[string]entity, error) {
	lines := strings.Split(strings.TrimSpace(string(source)), "\n")
	graph := map[string]entity{}
	for i := 0; i < len(lines); {
		parts := strings.Fields(lines[i])
		if len(parts) == 0 || parts[0] != "en" {
			i++
			continue
		}
		if len(parts) != 5 {
			return nil, fmt.Errorf("go_projection.malformed_entity")
		}
		count, err := strconv.Atoi(parts[4])
		if err != nil {
			return nil, fmt.Errorf("go_projection.malformed_entity")
		}
		e := entity{parts[1], parts[2], map[string][]string{}}
		i++
		for n := 0; n < count; n++ {
			for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				i++
			}
			if i >= len(lines) {
				return nil, fmt.Errorf("go_projection.truncated_field")
			}
			p := strings.Fields(lines[i])
			if len(p) < 3 || p[0] != "fi" {
				return nil, fmt.Errorf("go_projection.malformed_field:%q", lines[i])
			}
			key := p[1]
			value := []string{strings.Join(p[2:], " ")}
			i++
			if p[2] == "li" {
				if len(p) != 4 {
					return nil, fmt.Errorf("go_projection.malformed_list")
				}
				size, x := strconv.Atoi(p[3])
				if x != nil || i+size > len(lines) {
					return nil, fmt.Errorf("go_projection.malformed_list")
				}
				for j := 0; j < size; j++ {
					value = append(value, strings.TrimSpace(lines[i]))
					i++
				}
			}
			if _, exists := e.fields[key]; exists {
				return nil, fmt.Errorf("go_projection.duplicate_field")
			}
			e.fields[key] = value
		}
		if _, exists := graph[e.id]; exists {
			return nil, fmt.Errorf("go_projection.duplicate_entity")
		}
		graph[e.id] = e
	}
	return graph, nil
}
func scalar(e entity, field string) (string, error) {
	v, ok := e.fields[field]
	if !ok || len(v) != 1 {
		return "", fmt.Errorf("go_projection.missing_field")
	}
	return v[0], nil
}
func ref(e entity, field string) (string, error) {
	v, err := scalar(e, field)
	if err != nil {
		return "", err
	}
	p := strings.Fields(v)
	if len(p) != 2 || p[0] != "rf" {
		return "", fmt.Errorf("go_projection.invalid_reference")
	}
	return p[1], nil
}
func refs(e entity, field string) ([]string, error) {
	v, ok := e.fields[field]
	if !ok || len(v) < 1 {
		return nil, fmt.Errorf("go_projection.missing_field")
	}
	p := strings.Fields(v[0])
	if len(p) != 2 || p[0] != "li" {
		return nil, fmt.Errorf("go_projection.invalid_reference_list")
	}
	count, err := strconv.Atoi(p[1])
	if err != nil || len(v) != count+1 {
		return nil, fmt.Errorf("go_projection.invalid_reference_list")
	}
	out := make([]string, count)
	for i := range out {
		p = strings.Fields(v[i+1])
		if len(p) != 2 || p[0] != "rf" {
			return nil, fmt.Errorf("go_projection.invalid_reference")
		}
		out[i] = p[1]
	}
	return out, nil
}
func text(e entity, field string) (string, error) {
	v, err := scalar(e, field)
	if err != nil {
		return "", err
	}
	p := strings.Fields(v)
	if len(p) != 2 || p[0] != "by" {
		return "", fmt.Errorf("go_projection.invalid_text")
	}
	if p[1] == "-" {
		return "", nil
	}
	b, err := hex.DecodeString(p[1])
	if err != nil || !utf8.Valid(b) {
		return "", fmt.Errorf("go_projection.invalid_utf8")
	}
	return string(b), nil
}
