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

// ValidateSliceConstructElementCount enforces the language-neutral v34 bound.
func ValidateSliceConstructElementCount(count int) error {
	if count < 0 || count > 512 {
		return fmt.Errorf("slice construct element count %d outside 0..512", count)
	}
	return nil
}

func Declarations(version int) ([]Schema, error) {
	if version < 2 || version > 57 {
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
	if version >= 6 {
		schemas = append(schemas,
			Schema{0x9060, "FunctionCall", []Field{field(0x9600, "function_call.callee", 5, 0x9011, 0), field(0x9601, "function_call.arguments", 5, 0, 2)}},
		)
	}
	if version >= 7 {
		schemas = append(schemas,
			Schema{0x9070, "IntegerLiteral", []Field{field(0x9700, "integer_literal.bits", 2, 0, 0), field(0x9701, "integer_literal.type", 5, 0x9010, 0)}},
		)
	}
	if version >= 8 {
		schemas = append(schemas,
			Schema{0x9080, "Block", []Field{field(0x9800, "block.statements", 5, 0, 2)}},
			Schema{0x9081, "Return", []Field{field(0x9810, "return.values", 5, 0, 2)}},
		)
	}
	if version >= 9 {
		schemas = append(schemas,
			Schema{0x9090, "IntegerMultiply", []Field{field(0x9900, "integer_multiply.left", 5, 0, 0), field(0x9901, "integer_multiply.right", 5, 0, 0), field(0x9902, "integer_multiply.type", 5, 0x9010, 0)}},
		)
	}
	if version >= 10 {
		schemas = append(schemas,
			Schema{0x90a0, "IntegerSubtract", []Field{field(0x9a00, "integer_subtract.left", 5, 0, 0), field(0x9a01, "integer_subtract.right", 5, 0, 0), field(0x9a02, "integer_subtract.type", 5, 0x9010, 0)}},
		)
	}
	if version >= 11 {
		schemas = append(schemas,
			Schema{0x90b0, "BooleanLiteral", []Field{field(0x9b00, "boolean_literal.value", 1, 0, 0)}},
			Schema{0x90b1, "BooleanAnd", []Field{field(0x9b10, "boolean_and.left", 5, 0, 0), field(0x9b11, "boolean_and.right", 5, 0, 0)}},
		)
	}
	if version >= 13 {
		schemas = append(schemas,
			Schema{0x90c0, "If", []Field{field(0x9c00, "if.condition", 5, 0, 0), field(0x9c01, "if.then", 5, 0x9080, 0), field(0x9c02, "if.else", 5, 0x9080, 0)}},
			Schema{0x90c1, "BooleanOr", []Field{field(0x9c10, "boolean_or.left", 5, 0, 0), field(0x9c11, "boolean_or.right", 5, 0, 0)}},
			Schema{0x90c2, "StringEqual", []Field{field(0x9c20, "string_equal.left", 5, 0, 0), field(0x9c21, "string_equal.right", 5, 0, 0)}},
			Schema{0x90c3, "StringConcat", []Field{field(0x9c30, "string_concat.left", 5, 0, 0), field(0x9c31, "string_concat.right", 5, 0, 0)}},
		)
	}
	if version >= 15 {
		schemas = append(schemas,
			Schema{0x90d0, "LocalBinding", []Field{field(0x9d00, "local_binding.name", 4, 0, 0), field(0x9d01, "local_binding.type", 5, 0, 0), field(0x9d02, "local_binding.initializer", 5, 0, 0)}},
			Schema{0x90d1, "BindLocal", []Field{field(0x9d10, "bind_local.binding", 5, 0x90d0, 0)}},
			Schema{0x90d2, "LocalRead", []Field{field(0x9d20, "local_read.binding", 5, 0x90d0, 0)}},
		)
	}
	if version >= 18 {
		schemas = append(schemas,
			Schema{0x90e0, "MutablePlace", []Field{field(0x9e00, "mutable_place.name", 4, 0, 0), field(0x9e01, "mutable_place.type", 5, 0, 0), field(0x9e02, "mutable_place.initializer", 5, 0, 0)}},
			Schema{0x90e1, "DeclarePlace", []Field{field(0x9e10, "declare_place.place", 5, 0x90e0, 0)}},
			Schema{0x90e2, "PlaceRead", []Field{field(0x9e20, "place_read.place", 5, 0x90e0, 0)}},
			Schema{0x90e3, "AssignPlace", []Field{field(0x9e30, "assign_place.place", 5, 0x90e0, 0), field(0x9e31, "assign_place.value", 5, 0, 0)}},
			Schema{0x90e4, "While", []Field{field(0x9e40, "while.condition", 5, 0, 0), field(0x9e41, "while.body", 5, 0x9080, 0)}},
		)
	}
	if version >= 19 {
		schemas = append(schemas,
			Schema{0x90f0, "When", []Field{field(0x9f00, "when.condition", 5, 0, 0), field(0x9f01, "when.body", 5, 0x9080, 0)}},
		)
	}
	if version >= 20 {
		schemas = append(schemas,
			Schema{0x90f1, "EffectInvoke", []Field{field(0x9f10, "effect_invoke.effect", 5, 0x15, 0), field(0x9f11, "effect_invoke.arguments", 5, 0, 2)}},
		)
	}
	if version >= 21 {
		schemas = append(schemas,
			Schema{0x90f2, "FixedArrayType", []Field{field(0x9f20, "fixed_array.element_type", 5, 0, 0), field(0x9f21, "fixed_array.length", 2, 0, 0)}},
			Schema{0x90f3, "FixedArrayConstruct", []Field{field(0x9f30, "fixed_array_construct.type", 5, 0x90f2, 0), field(0x9f31, "fixed_array_construct.values", 5, 0, 2)}},
			Schema{0x90f4, "IndexRead", []Field{field(0x9f40, "index_read.collection", 5, 0, 0), field(0x9f41, "index_read.index", 5, 0, 0)}},
		)
	}
	if version >= 22 {
		schemas = append(schemas,
			Schema{0x90f5, "IterationBinding", []Field{field(0x9f50, "iteration_binding.name", 4, 0, 0), field(0x9f51, "iteration_binding.type", 5, 0, 0)}},
			Schema{0x90f6, "IterationBindingRead", []Field{field(0x9f60, "iteration_binding_read.binding", 5, 0x90f5, 0)}},
			Schema{0x90f7, "Fold", []Field{field(0x9f70, "fold.collection", 5, 0, 0), field(0x9f71, "fold.initial", 5, 0, 0), field(0x9f72, "fold.accumulator", 5, 0x90f5, 0), field(0x9f73, "fold.element", 5, 0x90f5, 0), field(0x9f74, "fold.body", 5, 0, 0)}},
		)
	}
	if version >= 23 {
		schemas = append(schemas,
			Schema{0x90f8, "SliceType", []Field{field(0x9f80, "slice.element_type", 5, 0, 0)}},
		)
	}
	if version >= 24 {
		schemas = append(schemas,
			Schema{0x90f9, "CollectionLength", []Field{field(0x9f90, "collection_length.collection", 5, 0, 0)}},
			Schema{0x90fa, "DynamicIndexRead", []Field{field(0x9fa0, "dynamic_index_read.collection", 5, 0, 0), field(0x9fa1, "dynamic_index_read.index", 5, 0, 0)}},
		)
	}
	if version >= 25 {
		schemas = append(schemas,
			Schema{0x90fb, "CollectionAppend", []Field{field(0x9fb0, "collection_append.collection", 5, 0, 0), field(0x9fb1, "collection_append.value", 5, 0, 0)}},
			Schema{0x90fc, "CollectionUpdate", []Field{field(0x9fc0, "collection_update.collection", 5, 0, 0), field(0x9fc1, "collection_update.index", 5, 0, 0), field(0x9fc2, "collection_update.value", 5, 0, 0)}},
		)
	}
	if version >= 26 {
		schemas = append(schemas,
			Schema{0xa000, "ReceiverBinding", []Field{field(0xa0000, "receiver.name", 4, 0, 0), field(0xa0001, "receiver.type", 5, 0, 0)}},
			Schema{0xa001, "ReceiverRead", []Field{field(0xa0010, "receiver_read.receiver", 5, 0xa000, 0)}},
			Schema{0xa002, "Method", []Field{field(0xa0020, "method.name", 4, 0, 0), field(0xa0021, "method.receiver", 5, 0xa000, 0), field(0xa0022, "method.params", 5, 0x9012, 2), field(0xa0023, "method.result", 5, 0, 0), field(0xa0024, "method.body", 5, 0, 0)}},
			Schema{0xa003, "MethodCall", []Field{field(0xa0030, "call.receiver", 5, 0, 0), field(0xa0031, "call.method", 5, 0xa002, 0), field(0xa0032, "call.arguments", 5, 0, 2)}},
			Schema{0xa004, "StateTransitionType", []Field{field(0xa0040, "transition_type.state", 5, 0, 0), field(0xa0041, "transition_type.result", 5, 0, 0)}},
			Schema{0xa005, "StateTransition", []Field{field(0xa0050, "transition.type", 5, 0xa004, 0), field(0xa0051, "transition.state", 5, 0, 0), field(0xa0052, "transition.result", 5, 0, 0)}},
			Schema{0xa006, "TransitionState", []Field{field(0xa0060, "state.value", 5, 0, 0)}},
			Schema{0xa007, "TransitionResult", []Field{field(0xa0070, "result.value", 5, 0, 0)}},
		)
	}
	if version >= 27 {
		schemas = append(schemas,
			Schema{0xa010, "InterfaceType", []Field{field(0xa0100, "interface.name", 4, 0, 0), field(0xa0101, "interface.requirements", 5, 0xa011, 2)}},
			Schema{0xa011, "MethodRequirement", []Field{field(0xa0110, "requirement.name", 4, 0, 0), field(0xa0111, "requirement.parameters", 5, 0, 2), field(0xa0112, "requirement.result", 5, 0, 0)}},
			Schema{0xa012, "SatisfactionWitness", []Field{field(0xa0120, "witness.concrete_type", 5, 0, 0), field(0xa0121, "witness.interface_type", 5, 0xa010, 0), field(0xa0122, "witness.methods", 5, 0xa002, 2)}},
			Schema{0xa013, "InterfaceValue", []Field{field(0xa0130, "interface_value.type", 5, 0xa010, 0), field(0xa0131, "interface_value.value", 5, 0, 0), field(0xa0132, "interface_value.witness", 5, 0xa012, 0)}},
			Schema{0xa014, "DynamicMethodCall", []Field{field(0xa0140, "dynamic_call.receiver", 5, 0, 0), field(0xa0141, "dynamic_call.requirement", 5, 0xa011, 0), field(0xa0142, "dynamic_call.arguments", 5, 0, 2)}},
		)
	}
	if version >= 28 {
		schemas = append(schemas,
			Schema{0xa020, "FunctionType", []Field{field(0xa0200, "function_type.parameters", 5, 0, 2), field(0xa0201, "function_type.result", 5, 0, 0)}},
			Schema{0xa021, "CaptureBinding", []Field{field(0xa0210, "capture.name", 4, 0, 0), field(0xa0211, "capture.type", 5, 0, 0), field(0xa0212, "capture.value", 5, 0, 0)}},
			Schema{0xa022, "CaptureRead", []Field{field(0xa0220, "capture_read.capture", 5, 0xa021, 0)}},
			Schema{0xa023, "ClosureConstruct", []Field{field(0xa0230, "closure.type", 5, 0xa020, 0), field(0xa0231, "closure.parameters", 5, 0x9012, 2), field(0xa0232, "closure.captures", 5, 0xa021, 2), field(0xa0233, "closure.body", 5, 0, 0)}},
			Schema{0xa024, "IndirectCall", []Field{field(0xa0240, "indirect_call.callee", 5, 0, 0), field(0xa0241, "indirect_call.arguments", 5, 0, 2)}},
		)
	}
	if version >= 29 {
		schemas = append(schemas,
			Schema{0xa030, "MutableCaptureBinding", []Field{field(0xa0300, "mutable_capture.name", 4, 0, 0), field(0xa0301, "mutable_capture.type", 5, 0, 0), field(0xa0302, "mutable_capture.initial", 5, 0, 0)}},
			Schema{0xa031, "MutableCaptureRead", []Field{field(0xa0310, "mutable_capture_read.capture", 5, 0xa030, 0)}},
			Schema{0xa032, "CaptureUpdate", []Field{field(0xa0320, "capture_update.capture", 5, 0xa030, 0), field(0xa0321, "capture_update.value", 5, 0, 0)}},
			Schema{0xa033, "Sequence", []Field{field(0xa0330, "sequence.steps", 5, 0, 2), field(0xa0331, "sequence.result", 5, 0, 0)}},
			Schema{0xa034, "MutableClosureConstruct", []Field{field(0xa0340, "mutable_closure.type", 5, 0xa020, 0), field(0xa0341, "mutable_closure.parameters", 5, 0x9012, 2), field(0xa0342, "mutable_closure.captures", 5, 0xa030, 2), field(0xa0343, "mutable_closure.body", 5, 0, 0)}},
			Schema{0xa035, "StatefulIndirectCall", []Field{field(0xa0350, "stateful_call.callee", 5, 0, 0), field(0xa0351, "stateful_call.arguments", 5, 0, 2)}},
		)
	}
	if version >= 30 {
		schemas = append(schemas,
			Schema{0xa040, "MapType", []Field{field(0xa0400, "map.key", 5, 0, 0), field(0xa0401, "map.value", 5, 0, 0)}},
			Schema{0xa041, "EmptyMap", []Field{field(0xa0410, "empty_map.type", 5, 0xa040, 0)}},
			Schema{0xa042, "MapLookup", []Field{field(0xa0420, "map_lookup.map", 5, 0, 0), field(0xa0421, "map_lookup.key", 5, 0, 0)}},
			Schema{0xa043, "MapUpdate", []Field{field(0xa0430, "map_update.map", 5, 0, 0), field(0xa0431, "map_update.key", 5, 0, 0), field(0xa0432, "map_update.value", 5, 0, 0)}},
		)
	}
	if version >= 31 {
		schemas = append(schemas,
			Schema{0xa050, "OptionType", []Field{field(0xa0500, "option.value_type", 5, 0, 0)}},
			Schema{0xa051, "OptionNone", []Field{field(0xa0510, "option_none.type", 5, 0xa050, 0)}},
			Schema{0xa052, "OptionSome", []Field{field(0xa0520, "option_some.type", 5, 0xa050, 0), field(0xa0521, "option_some.value", 5, 0, 0)}},
		)
	}
	if version >= 32 {
		schemas = append(schemas,
			Schema{0xa060, "VariantBinding", []Field{field(0xa0600, "variant_binding.name", 4, 0, 0), field(0xa0601, "variant_binding.type", 5, 0, 0)}},
			Schema{0xa061, "VariantBindingRead", []Field{field(0xa0610, "variant_binding_read.binding", 5, 0xa060, 0)}},
			Schema{0xa062, "ResultMatch", []Field{field(0xa0620, "result_match.value", 5, 0, 0), field(0xa0621, "result_match.ok_binding", 5, 0xa060, 0), field(0xa0622, "result_match.ok_body", 5, 0x9080, 0), field(0xa0623, "result_match.error_binding", 5, 0xa060, 0), field(0xa0624, "result_match.error_body", 5, 0x9080, 0)}},
			Schema{0xa063, "OptionMatch", []Field{field(0xa0630, "option_match.value", 5, 0, 0), field(0xa0631, "option_match.none_body", 5, 0x9080, 0), field(0xa0632, "option_match.some_binding", 5, 0xa060, 0), field(0xa0633, "option_match.some_body", 5, 0x9080, 0)}},
			Schema{0xa064, "BytesLiteral", []Field{field(0xa0640, "bytes_literal.value", 4, 0, 0)}},
			Schema{0xa065, "BytesEqual", []Field{field(0xa0650, "bytes_equal.left", 5, 0, 0), field(0xa0651, "bytes_equal.right", 5, 0, 0)}},
		)
	}
	if version >= 33 {
		schemas = append(schemas,
			Schema{0xa066, "SliceRemove", []Field{field(0xa0660, "slice_remove.slice", 5, 0, 0), field(0xa0661, "slice_remove.index", 5, 0, 0)}},
			Schema{0xa067, "MapRemove", []Field{field(0xa0670, "map_remove.map", 5, 0, 0), field(0xa0671, "map_remove.key", 5, 0, 0)}},
		)
	}
	if version >= 34 {
		schemas = append(schemas,
			Schema{0xa068, "SliceConstruct", []Field{field(0xa0680, "slice_construct.type", 5, 0x90f8, 0), field(0xa0681, "slice_construct.elements", 5, 0, 2)}},
		)
	}
	if version >= 35 {
		schemas = append(schemas,
			Schema{0xa044, "MapLookupOption", []Field{field(0xa0440, "map_lookup_option.map", 5, 0, 0), field(0xa0441, "map_lookup_option.key", 5, 0, 0), field(0xa0442, "map_lookup_option.type", 5, 0xa050, 0)}},
		)
	}
	if version >= 36 {
		schemas = append(schemas,
			Schema{0xa069, "BooleanNot", []Field{field(0xa0690, "boolean_not.value", 5, 0, 0)}},
		)
	}
	if version >= 37 {
		schemas = append(schemas,
			Schema{0xa06a, "UnitType", nil},
			Schema{0xa06b, "UnitValue", []Field{field(0xa06b0, "unit_value.type", 5, 0xa06a, 0)}},
		)
	}
	if version >= 38 {
		schemas = append(schemas,
			Schema{0xa06c, "Evaluate", []Field{field(0xa06c0, "evaluate.value", 5, 0, 0)}},
		)
	}
	if version >= 39 {
		schemas = append(schemas,
			Schema{0xa06d, "NativeInvocation", []Field{
				field(0xa06d0, "native_invocation.language", 4, 0, 0),
				field(0xa06d1, "native_invocation.callable", 4, 0, 0),
				field(0xa06d2, "native_invocation.signature", 4, 0, 0),
				field(0xa06d3, "native_invocation.arguments", 5, 0, 2),
				field(0xa06d4, "native_invocation.result_type", 5, 0, 0),
			}},
		)
	}
	if version >= 40 {
		schemas = append(schemas,
			Schema{0xa06e, "NativeMethodInvocation", []Field{
				field(0xa06e0, "native_method_invocation.language", 4, 0, 0),
				field(0xa06e1, "native_method_invocation.callable", 4, 0, 0),
				field(0xa06e2, "native_method_invocation.signature", 4, 0, 0),
				field(0xa06e3, "native_method_invocation.receiver", 5, 0, 0),
				field(0xa06e4, "native_method_invocation.arguments", 5, 0, 2),
				field(0xa06e5, "native_method_invocation.result_type", 5, 0, 0),
			}},
		)
	}
	if version >= 41 {
		schemas = append(schemas,
			Schema{0xa06f, "ProductType", []Field{field(0xa06f0, "product.item_types", 5, 0, 2)}},
			Schema{0xa070, "ProductProject", []Field{field(0xa0700, "product_project.product", 5, 0, 0), field(0xa0701, "product_project.type", 5, 0xa06f, 0), field(0xa0702, "product_project.index", 2, 0, 0), field(0xa0703, "product_project.item_type", 5, 0, 0)}},
			Schema{0xa071, "NativeType", []Field{field(0xa0710, "native_type.language", 4, 0, 0), field(0xa0711, "native_type.spelling", 4, 0, 0)}},
		)
	}
	if version >= 43 {
		schemas = append(schemas,
			Schema{0xa072, "NativeFieldRead", []Field{
				field(0xa0720, "native_field_read.language", 4, 0, 0),
				field(0xa0721, "native_field_read.field", 4, 0, 0),
				field(0xa0722, "native_field_read.receiver", 5, 0, 0),
				field(0xa0723, "native_field_read.result_type", 5, 0, 0),
			}},
			Schema{0xa073, "NativeBindingRead", []Field{
				field(0xa0730, "native_binding_read.language", 4, 0, 0),
				field(0xa0731, "native_binding_read.binding", 4, 0, 0),
				field(0xa0732, "native_binding_read.result_type", 5, 0, 0),
			}},
		)
	}
	if version >= 45 {
		schemas = append(schemas, Schema{0xa074, "NativeDefaultValue", []Field{
			field(0xa0740, "native_default.language", 4, 0, 0),
			field(0xa0741, "native_default.type", 5, 0xa071, 0),
		}})
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
	if version >= 26 {
		return emitOrdered(output, version, schemas, fields)
	}
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

type declaration struct {
	id     uint64
	schema *Schema
	field  *Field
}

func emitOrdered(output io.Writer, version int, schemas []Schema, fields []Field) error {
	declarations := make([]declaration, 0, len(schemas)+len(fields))
	for index := range schemas {
		declarations = append(declarations, declaration{id: schemas[index].ID, schema: &schemas[index]})
	}
	for index := range fields {
		declarations = append(declarations, declaration{id: fields[index].ID, field: &fields[index]})
	}
	sort.Slice(declarations, func(i, j int) bool { return declarations[i].id < declarations[j].id })
	fmt.Fprintf(output, "# Generated construction projection for Core Execution Semantics v%d.\nve 1\nmo %s\nrv %s\npc 1\n%s\nec %d\n\n", version, id(0x9000), id(uint64(0x9000+version)), id(0x9001), 1+len(declarations))
	fmt.Fprintf(output, "en %s %s %d 2\nfi %s by %s\nfi %s li %d\n", id(0x9000), id(0x12), version, id(0x120), text(fmt.Sprintf("core-execution-v%d", version)), id(0x122), len(declarations))
	for _, item := range declarations {
		fmt.Fprintf(output, "rf %s\n", id(item.id))
	}
	for _, item := range declarations {
		if item.schema != nil {
			emitSchema(output, *item.schema)
		} else {
			emitField(output, *item.field)
		}
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
