// Package executionmodule emits versioned Core Execution semantic modules.
package executionmodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

type Field struct {
	ID                 uint64
	Name               string
	Kind, Schema, Card uint64
}
type Schema struct {
	ID     uint64
	Name   string
	Fields []Field
}

func field(id uint64, name string, kind, schema, card uint64) Field {
	return Field{id, name, kind, schema, card}
}

func Declarations(version int) ([]Schema, error) {
	if version < 2 || version > 6 {
		return nil, fmt.Errorf("unsupported Core Execution version %d", version)
	}
	schemas := []Schema{
		{0x9010, "IntegerType", []Field{field(0x9100, "integer.width", 2, 0, 0), field(0x9101, "integer.signed", 1, 0, 0), field(0x9102, "integer.overflow", 2, 0, 0)}},
		{0x9011, "Function", []Field{field(0x9110, "function.name", 4, 0, 0), field(0x9111, "function.parameters", 5, 0x9012, 2), field(0x9112, "function.result_type", 5, 0, 0), field(0x9113, "function.body", 5, 0, 0)}},
		{0x9012, "Parameter", []Field{field(0x9120, "parameter.name", 4, 0, 0), field(0x9121, "parameter.type", 5, 0, 0), field(0x9122, "parameter.index", 2, 0, 0)}},
		{0x9013, "ParameterRead", []Field{field(0x9130, "parameter_read.parameter", 5, 0x9012, 0)}},
		{0x9014, "IntegerAdd", []Field{field(0x9140, "integer_add.left", 5, 0, 0), field(0x9141, "integer_add.right", 5, 0, 0), field(0x9142, "integer_add.type", 5, 0x9010, 0)}},
		{0x9015, "ExecutableProgram", []Field{field(0x9150, "program.functions", 5, 0x9011, 2), field(0x9151, "program.entry", 5, 0x9011, 0)}},
		{0x9020, "BooleanType", nil},
		{0x9021, "IntegerLessEqual", []Field{field(0x9160, "integer_less_equal.left", 5, 0, 0), field(0x9161, "integer_less_equal.right", 5, 0, 0), field(0x9162, "integer_less_equal.operand_type", 5, 0x9010, 0)}},
	}
	if version == 3 {
		schemas = append(schemas, recordSchemas()...)
	}
	if version >= 4 {
		schemas = append(schemas, recordSchemas()...)
		schemas = append(schemas,
			Schema{0x9040, "StringType", nil},
			Schema{0x9041, "BytesType", nil},
			Schema{0x9042, "ResultType", []Field{field(0x9400, "result.ok_type", 5, 0, 0), field(0x9401, "result.error_type", 5, 0, 0)}},
			Schema{0x9043, "ResultOk", []Field{field(0x9410, "result_ok.type", 5, 0x9042, 0), field(0x9411, "result_ok.value", 5, 0, 0)}},
			Schema{0x9044, "ResultError", []Field{field(0x9420, "result_error.type", 5, 0x9042, 0), field(0x9421, "result_error.error", 5, 0, 0)}},
		)
	}
	if version >= 5 {
		schemas = append(schemas,
			Schema{0x9050, "StringLiteral", []Field{field(0x9500, "string_literal.value", 4, 0, 0)}},
			Schema{0x9051, "StringIsEmpty", []Field{field(0x9510, "string_is_empty.value", 5, 0, 0)}},
			Schema{0x9052, "Conditional", []Field{field(0x9520, "conditional.condition", 5, 0, 0), field(0x9521, "conditional.then", 5, 0, 0), field(0x9522, "conditional.else", 5, 0, 0)}},
		)
	}
	if version == 6 {
		schemas = append(schemas,
			Schema{0x9060, "FunctionCall", []Field{field(0x9600, "function_call.callee", 5, 0x9011, 0), field(0x9601, "function_call.arguments", 5, 0, 2)}},
		)
	}
	return schemas, nil
}

func recordSchemas() []Schema {
	return []Schema{
		{0x9030, "RecordType", []Field{field(0x9300, "record.name", 4, 0, 0), field(0x9301, "record.fields", 5, 0x9031, 2)}},
		{0x9031, "RecordField", []Field{field(0x9310, "record_field.name", 4, 0, 0), field(0x9311, "record_field.type", 5, 0, 0), field(0x9312, "record_field.index", 2, 0, 0)}},
		{0x9032, "FieldRead", []Field{field(0x9320, "field_read.record", 5, 0, 0), field(0x9321, "field_read.field", 5, 0x9031, 0)}},
		{0x9033, "RecordConstruct", []Field{field(0x9330, "record_construct.type", 5, 0x9030, 0), field(0x9331, "record_construct.values", 5, 0, 2)}},
	}
}

func Emit(output io.Writer, version int) error {
	schemas, err := Declarations(version)
	if err != nil {
		return err
	}
	var fields []Field
	for _, schema := range schemas {
		fields = append(fields, schema.Fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].ID < fields[j].ID })
	fmt.Fprintf(output, "# Generated construction projection for Core Execution Semantics v%d.\nve 1\nmo %s\nrv %s\npc 1\n%s\nec %d\n\n", version, id(0x9000), id(uint64(0x9000+version)), id(0x9001), 1+len(schemas)+len(fields))
	fmt.Fprintf(output, "en %s %s %d 2\nfi %s by %s\nfi %s li %d\n", id(0x9000), id(0x12), version, id(0x120), text(fmt.Sprintf("core-execution-v%d", version)), id(0x122), len(schemas)+len(fields))
	for _, schema := range schemas {
		fmt.Fprintf(output, "rf %s\n", id(schema.ID))
	}
	for _, field := range fields {
		fmt.Fprintf(output, "rf %s\n", id(field.ID))
	}
	for _, schema := range schemas {
		emitSchema(output, schema)
	}
	for _, field := range fields {
		emitField(output, field)
	}
	return nil
}

func emitSchema(output io.Writer, schema Schema) {
	fmt.Fprintf(output, "\nen %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(schema.ID), id(0x10), id(0x100), text(schema.Name), id(0x101), len(schema.Fields))
	for _, field := range schema.Fields {
		fmt.Fprintf(output, "rf %s\n", id(field.ID))
	}
}
func emitField(output io.Writer, field Field) {
	count := 1
	if field.Schema != 0 {
		count = 2
	}
	fmt.Fprintf(output, "\nen %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n", id(field.ID), id(0x11), id(0x110), text(field.Name), id(0x111), count, id(0x2000), field.Kind)
	if field.Schema != 0 {
		fmt.Fprintf(output, "fi %s rf %s\n", id(0x2001), id(field.Schema))
	}
	fmt.Fprintf(output, "fi %s uu %d\nfi %s uu 1\n", id(0x112), field.Card, id(0x113))
}
func id(value uint64) string   { return fmt.Sprintf("%032x", value) }
func text(value string) string { return hex.EncodeToString([]byte(value)) }
