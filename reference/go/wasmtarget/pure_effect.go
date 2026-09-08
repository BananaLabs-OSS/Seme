package wasmtarget

import (
	"fmt"

	"seme.local/reference/wire"
)

func certifyPureEffectFunction(graph wire.Envelope, bodyID wire.ID, parameters []pureValueType, result pureValueType, parameterTypes map[wire.ID]string, locals map[wire.ID]byte, abi PureABI) ([]byte, PureABI, error) {
	abi.Fidelity = "adapted"
	abi.RequiredCapabilities = []string{"observability.log"}
	effects := bySchema(graph, 0x15)
	capabilities := bySchema(graph, 0x16)
	if len(effects) != 1 || len(capabilities) != 1 {
		return nil, PureABI{}, fmt.Errorf("wasm.effect_cardinality")
	}
	nameValue, nameErr := field(effects[0], 0x150)
	capabilityValue, capabilityErr := field(effects[0], 0x151)
	capabilityName, capabilityNameErr := field(capabilities[0], 0x160)
	if nameErr != nil || capabilityErr != nil || capabilityNameErr != nil || nameValue.Tag != 5 || capabilityValue.Tag != 6 || capabilityValue.Reference != capabilities[0].ID || capabilityName.Tag != 5 || string(nameValue.Bytes) != "observability.log" || string(capabilityName.Bytes) != "observability.log" {
		return nil, PureABI{}, fmt.Errorf("wasm.unsupported_effect")
	}
	block, ok := graph.Entities[bodyID]
	statements, statementsErr := field(block, 0x9800)
	if !ok || block.Schema != identity(0x9080) || statementsErr != nil || statements.Tag != 7 || len(statements.List) < 2 {
		return nil, PureABI{}, fmt.Errorf("wasm.effect_block")
	}
	budget := 4096
	used := map[byte]bool{}
	var instructions []byte
	for index, item := range statements.List {
		if item.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.effect_statement")
		}
		statement, exists := graph.Entities[item.Reference]
		if !exists {
			return nil, PureABI{}, fmt.Errorf("wasm.effect_statement")
		}
		if index == len(statements.List)-1 {
			if statement.Schema != identity(0x9081) {
				return nil, PureABI{}, fmt.Errorf("wasm.effect_missing_return")
			}
			values, err := field(statement, 0x9810)
			if err != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
				return nil, PureABI{}, fmt.Errorf("wasm.pure_return_values")
			}
			if err := validatePureExpression(graph, values.List[0].Reference, result.name, parameterTypes, map[wire.ID]bool{}, &budget); err != nil {
				return nil, PureABI{}, err
			}
			var resultCode []byte
			if result.name == "bool" {
				resultCode, err = lowerHelperBoolean(graph, values.List[0].Reference, locals, used, map[wire.ID]bool{}, &budget)
			} else if result.name == "i64" {
				resultCode, err = lowerHelperInteger(graph, values.List[0].Reference, locals, used, map[wire.ID]bool{}, &budget)
			} else {
				err = fmt.Errorf("wasm.effect_result_type")
			}
			if err != nil {
				return nil, PureABI{}, err
			}
			instructions = append(instructions, resultCode...)
			continue
		}
		if statement.Schema != identity(0x90f1) {
			return nil, PureABI{}, fmt.Errorf("wasm.effect_statement")
		}
		effectValue, effectErr := field(statement, 0x9f10)
		arguments, argumentsErr := field(statement, 0x9f11)
		if effectErr != nil || argumentsErr != nil || effectValue.Tag != 6 || effectValue.Reference != effects[0].ID || arguments.Tag != 7 || len(arguments.List) != 1 || arguments.List[0].Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.effect_invocation")
		}
		if err := validatePureExpression(graph, arguments.List[0].Reference, "bool", parameterTypes, map[wire.ID]bool{}, &budget); err != nil {
			return nil, PureABI{}, err
		}
		argumentCode, err := lowerHelperBoolean(graph, arguments.List[0].Reference, locals, used, map[wire.ID]bool{}, &budget)
		if err != nil {
			return nil, PureABI{}, err
		}
		instructions = append(instructions, argumentCode...)
		instructions = append(instructions, 0x10, 0x00, 0x04, 0x40, 0x00, 0x0b)
	}
	wasm, err := pureModule(parameters, result, instructions, abi, true)
	return wasm, abi, err
}
