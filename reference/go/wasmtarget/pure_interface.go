package wasmtarget

import (
	"bytes"
	"fmt"
	"sort"

	"seme.local/reference/wire"
)

type interfaceDispatchCase struct {
	witness wire.Entity
	method  wire.Entity
	code    []byte
}

// certifyPureInterfaceFunction realizes bounded interface dispatch without
// importing any source-language object model. A stable tag selects a canonical
// satisfaction witness; the witness, requirement, receiver type, and method
// signature are all certified before Wasm is emitted.
func certifyPureInterfaceFunction(graph wire.Envelope, program, function wire.Entity) ([]byte, PureABI, error) {
	parameters, err := field(function, 0x9111)
	if err == nil && parameters.Tag == 7 && len(parameters.List) == 3 {
		return certifyPureInterfaceDispatchFunction(graph, program, function)
	}
	return certifyPureInterfaceApplyFunction(graph, program, function)
}

func certifyPureInterfaceApplyFunction(graph wire.Envelope, program, function wire.Entity) ([]byte, PureABI, error) {
	parameters, err := field(function, 0x9111)
	if err != nil || parameters.Tag != 7 || len(parameters.List) != 2 {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_parameters")
	}
	parameterIDs := make([]wire.ID, 2)
	for index, item := range parameters.List {
		if item.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.interface_parameter")
		}
		parameter := graph.Entities[item.Reference]
		position, pErr := field(parameter, 0x9122)
		typeValue, tErr := field(parameter, 0x9121)
		if parameter.Schema != identity(0x9012) || pErr != nil || tErr != nil || position.Tag != 3 || position.Unsigned != uint64(index) || typeValue.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.interface_parameter")
		}
		parameterIDs[index] = parameter.ID
		if index == 0 {
			if graph.Entities[typeValue.Reference].Schema != identity(0xa010) {
				return nil, PureABI{}, fmt.Errorf("wasm.interface_parameter_type")
			}
		} else if scalar, scalarErr := pureType(graph, typeValue.Reference); scalarErr != nil || scalar.name != "i64" {
			return nil, PureABI{}, fmt.Errorf("wasm.interface_argument_type")
		}
	}
	interfaceTypeValue, _ := field(graph.Entities[parameters.List[0].Reference], 0x9121)
	interfaceType := graph.Entities[interfaceTypeValue.Reference]
	requirements, reqErr := field(interfaceType, 0xa0101)
	if reqErr != nil || requirements.Tag != 7 || len(requirements.List) != 1 || requirements.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_requirements")
	}
	requirement := graph.Entities[requirements.List[0].Reference]
	if err := validateI64Requirement(graph, requirement); err != nil {
		return nil, PureABI{}, err
	}
	result, resultErr := field(function, 0x9112)
	if resultErr != nil || result.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_result")
	}
	if scalar, scalarErr := pureType(graph, result.Reference); scalarErr != nil || scalar.name != "i64" {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_result")
	}
	body, _ := field(function, 0x9113)
	block := graph.Entities[body.Reference]
	statements, blockErr := field(block, 0x9800)
	if body.Tag != 6 || block.Schema != identity(0x9080) || blockErr != nil || statements.Tag != 7 || len(statements.List) != 1 || statements.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_entry_body")
	}
	ret := graph.Entities[statements.List[0].Reference]
	values, returnErr := field(ret, 0x9810)
	if ret.Schema != identity(0x9081) || returnErr != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_entry_return")
	}
	call := graph.Entities[values.List[0].Reference]
	receiver, rErr := field(call, 0xa0140)
	calledRequirement, qErr := field(call, 0xa0141)
	arguments, aErr := field(call, 0xa0142)
	if call.Schema != identity(0xa014) || rErr != nil || qErr != nil || aErr != nil || calledRequirement.Reference != requirement.ID || arguments.Tag != 7 || len(arguments.List) != 1 || arguments.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_dynamic_call")
	}
	entryLocals := map[wire.ID]byte{parameterIDs[0]: 0, parameterIDs[1]: 1}
	receiverCode, err := lowerTransitionI64(graph, receiver.Reference, entryLocals, nil, map[wire.ID]bool{})
	if err != nil || !bytes.Equal(receiverCode, []byte{0x20, 0}) {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_receiver_binding")
	}
	argumentCode, err := lowerTransitionI64(graph, arguments.List[0].Reference, entryLocals, nil, map[wire.ID]bool{})
	if err != nil || !bytes.Equal(argumentCode, []byte{0x20, 1}) {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_argument_binding")
	}

	var cases []interfaceDispatchCase
	for _, witness := range bySchema(graph, 0xa012) {
		concrete, cErr := field(witness, 0xa0120)
		contract, iErr := field(witness, 0xa0121)
		methods, mErr := field(witness, 0xa0122)
		if cErr != nil || iErr != nil || mErr != nil || contract.Reference != interfaceType.ID || methods.Tag != 7 || len(methods.List) != 1 || methods.List[0].Tag != 6 {
			if iErr == nil && contract.Tag == 6 && contract.Reference != interfaceType.ID {
				continue
			}
			return nil, PureABI{}, fmt.Errorf("wasm.interface_witness")
		}
		if err := validateSingleI64Record(graph, concrete.Reference); err != nil {
			return nil, PureABI{}, fmt.Errorf("wasm.interface_concrete_type")
		}
		method := graph.Entities[methods.List[0].Reference]
		code, err := certifyInterfaceMethod(graph, method, concrete.Reference, requirement)
		if err != nil {
			return nil, PureABI{}, err
		}
		cases = append(cases, interfaceDispatchCase{witness: witness, method: method, code: code})
	}
	if len(cases) == 0 || len(cases) > 32 {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_witness_cardinality")
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].witness.ID.String() < cases[j].witness.ID.String() })
	abi := PureABI{Contract: "seme.pure-abi/v1", Provider: "seme.function-v1", Target: "wasm32-pulp-reactor-v1", Fidelity: "exact", CanonicalModule: graph.Module.String(), CanonicalRevision: graph.Revision.String(), CanonicalProgram: program.ID.String(), Function: function.ID.String(), RequestSize: 24, ResponseSize: 8,
		Parameters: []PureABIField{{Index: 0, Type: "interface:record:i64", Offset: 0, Size: 16, Encoding: "witness-tag-u64-then-little-endian-i64"}, {Index: 1, Type: "i64", Offset: 16, Size: 8, Encoding: "little-endian-i64"}},
		Result:     PureABIField{Index: 0, Type: "i64", Offset: 0, Size: 8, Encoding: "little-endian-i64"}}
	return pureInterfaceModule(cases, abi)
}

func certifyPureInterfaceDispatchFunction(graph wire.Envelope, program, function wire.Entity) ([]byte, PureABI, error) {
	parameters, _ := field(function, 0x9111)
	parameterIDs := make([]wire.ID, 3)
	wantTypes := []string{"bool", "i64", "i64"}
	for index, item := range parameters.List {
		if item.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.interface_dispatch_parameter")
		}
		parameter := graph.Entities[item.Reference]
		typeValue, a := field(parameter, 0x9121)
		position, b := field(parameter, 0x9122)
		if parameter.Schema != identity(0x9012) || a != nil || b != nil || position.Unsigned != uint64(index) {
			return nil, PureABI{}, fmt.Errorf("wasm.interface_dispatch_parameter")
		}
		typeInfo, err := pureType(graph, typeValue.Reference)
		if err != nil || typeInfo.name != wantTypes[index] {
			return nil, PureABI{}, fmt.Errorf("wasm.interface_dispatch_parameter_type")
		}
		parameterIDs[index] = parameter.ID
	}
	result, _ := field(function, 0x9112)
	if scalar, err := pureType(graph, result.Reference); err != nil || scalar.name != "i64" {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_dispatch_result")
	}
	body, _ := field(function, 0x9113)
	block := graph.Entities[body.Reference]
	statements, _ := field(block, 0x9800)
	if block.Schema != identity(0x9080) || statements.Tag != 7 || len(statements.List) != 1 {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_dispatch_body")
	}
	branch := graph.Entities[statements.List[0].Reference]
	condition, a := field(branch, 0x9c00)
	thenBlock, b := field(branch, 0x9c01)
	elseBlock, c := field(branch, 0x9c02)
	if branch.Schema != identity(0x90c0) || a != nil || b != nil || c != nil {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_dispatch_if")
	}
	locals := map[wire.ID]byte{parameterIDs[0]: 0, parameterIDs[1]: 1, parameterIDs[2]: 2}
	conditionCode, err := lowerTransitionI64(graph, condition.Reference, locals, nil, map[wire.ID]bool{})
	if err != nil || !bytes.Equal(conditionCode, []byte{0x20, 0}) {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_dispatch_condition")
	}

	type selected struct {
		witness wire.ID
		callee  wire.Entity
	}
	readBranch := func(blockID wire.ID) (selected, error) {
		branchBlock := graph.Entities[blockID]
		items, err := field(branchBlock, 0x9800)
		if err != nil || items.Tag != 7 || len(items.List) != 1 {
			return selected{}, fmt.Errorf("wasm.interface_dispatch_branch")
		}
		ret := graph.Entities[items.List[0].Reference]
		values, err := field(ret, 0x9810)
		if ret.Schema != identity(0x9081) || err != nil || values.Tag != 7 || len(values.List) != 1 {
			return selected{}, fmt.Errorf("wasm.interface_dispatch_branch")
		}
		call := graph.Entities[values.List[0].Reference]
		calleeValue, x := field(call, 0x9600)
		args, y := field(call, 0x9601)
		if call.Schema != identity(0x9060) || x != nil || y != nil || args.Tag != 7 || len(args.List) != 2 {
			return selected{}, fmt.Errorf("wasm.interface_dispatch_call")
		}
		iface := graph.Entities[args.List[0].Reference]
		witness, wErr := field(iface, 0xa0132)
		concreteValue, vErr := field(iface, 0xa0131)
		if iface.Schema != identity(0xa013) || wErr != nil || vErr != nil {
			return selected{}, fmt.Errorf("wasm.interface_dispatch_value")
		}
		stateCode, stateErr := lowerTransitionI64(graph, concreteValue.Reference, locals, nil, map[wire.ID]bool{})
		valueCode, valueErr := lowerTransitionI64(graph, args.List[1].Reference, locals, nil, map[wire.ID]bool{})
		if stateErr != nil || valueErr != nil || !bytes.Equal(stateCode, []byte{0x20, 1}) || !bytes.Equal(valueCode, []byte{0x20, 2}) {
			return selected{}, fmt.Errorf("wasm.interface_dispatch_binding")
		}
		callee := graph.Entities[calleeValue.Reference]
		if callee.Schema != identity(0x9011) {
			return selected{}, fmt.Errorf("wasm.interface_dispatch_callee")
		}
		return selected{witness.Reference, callee}, nil
	}
	trueSelection, err := readBranch(thenBlock.Reference)
	if err != nil {
		return nil, PureABI{}, err
	}
	falseSelection, err := readBranch(elseBlock.Reference)
	if err != nil {
		return nil, PureABI{}, err
	}
	if trueSelection.callee.ID != falseSelection.callee.ID || trueSelection.witness == falseSelection.witness {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_dispatch_selection")
	}
	// Certify the independently declared interface call and every witness before
	// using the branch-selected witnesses as physical dispatch cases.
	if _, _, err := certifyPureInterfaceApplyFunction(graph, program, trueSelection.callee); err != nil {
		return nil, PureABI{}, err
	}
	var cases []interfaceDispatchCase
	for _, witnessID := range []wire.ID{falseSelection.witness, trueSelection.witness} {
		witness := graph.Entities[witnessID]
		concrete, _ := field(witness, 0xa0120)
		methods, _ := field(witness, 0xa0122)
		method := graph.Entities[methods.List[0].Reference]
		applyBody, _ := field(trueSelection.callee, 0x9113)
		applyBlock := graph.Entities[applyBody.Reference]
		applyStatements, _ := field(applyBlock, 0x9800)
		applyReturn := graph.Entities[applyStatements.List[0].Reference]
		applyValues, _ := field(applyReturn, 0x9810)
		dynamic := graph.Entities[applyValues.List[0].Reference]
		requirementID, _ := field(dynamic, 0xa0141)
		code, err := certifyInterfaceMethod(graph, method, concrete.Reference, graph.Entities[requirementID.Reference])
		if err != nil {
			return nil, PureABI{}, err
		}
		cases = append(cases, interfaceDispatchCase{witness: witness, method: method, code: code})
	}
	abi := PureABI{Contract: "seme.pure-abi/v1", Provider: "seme.function-v1", Target: "wasm32-pulp-reactor-v1", Fidelity: "exact", CanonicalModule: graph.Module.String(), CanonicalRevision: graph.Revision.String(), CanonicalProgram: program.ID.String(), Function: function.ID.String(), RequestSize: 17, ResponseSize: 8,
		Parameters: []PureABIField{{Index: 0, Type: "bool", Offset: 0, Size: 1, Encoding: "zero-or-one-byte"}, {Index: 1, Type: "i64", Offset: 1, Size: 8, Encoding: "little-endian-i64"}, {Index: 2, Type: "i64", Offset: 9, Size: 8, Encoding: "little-endian-i64"}}, Result: PureABIField{Index: 0, Type: "i64", Offset: 0, Size: 8, Encoding: "little-endian-i64"}}
	return pureInterfaceDispatchModule(cases, abi)
}

func validateI64Requirement(graph wire.Envelope, requirement wire.Entity) error {
	parameters, pErr := field(requirement, 0xa0111)
	result, rErr := field(requirement, 0xa0112)
	if requirement.Schema != identity(0xa011) || pErr != nil || rErr != nil || parameters.Tag != 7 || len(parameters.List) != 1 || parameters.List[0].Tag != 6 || result.Tag != 6 {
		return fmt.Errorf("wasm.interface_requirement_signature")
	}
	for _, typeID := range []wire.ID{parameters.List[0].Reference, result.Reference} {
		if scalar, err := pureType(graph, typeID); err != nil || scalar.name != "i64" {
			return fmt.Errorf("wasm.interface_requirement_signature")
		}
	}
	return nil
}

func certifyInterfaceMethod(graph wire.Envelope, method wire.Entity, concrete wire.ID, requirement wire.Entity) ([]byte, error) {
	name, nErr := field(method, 0xa0020)
	requirementName, qErr := field(requirement, 0xa0110)
	receiver, rErr := field(method, 0xa0021)
	parameters, pErr := field(method, 0xa0022)
	result, outErr := field(method, 0xa0023)
	body, bErr := field(method, 0xa0024)
	if method.Schema != identity(0xa002) || nErr != nil || qErr != nil || !bytes.Equal(name.Bytes, requirementName.Bytes) || rErr != nil || pErr != nil || outErr != nil || bErr != nil || parameters.Tag != 7 || len(parameters.List) != 1 || parameters.List[0].Tag != 6 || result.Tag != 6 || body.Tag != 6 {
		return nil, fmt.Errorf("wasm.interface_method_signature")
	}
	receiverBinding := graph.Entities[receiver.Reference]
	receiverType, receiverTypeErr := field(receiverBinding, 0xa0001)
	parameter := graph.Entities[parameters.List[0].Reference]
	parameterType, parameterTypeErr := field(parameter, 0x9121)
	position, positionErr := field(parameter, 0x9122)
	if receiverBinding.Schema != identity(0xa000) || receiverTypeErr != nil || receiverType.Reference != concrete || parameter.Schema != identity(0x9012) || parameterTypeErr != nil || positionErr != nil || position.Unsigned != 0 {
		return nil, fmt.Errorf("wasm.interface_method_signature")
	}
	for _, typeID := range []wire.ID{parameterType.Reference, result.Reference} {
		if scalar, err := pureType(graph, typeID); err != nil || scalar.name != "i64" {
			return nil, fmt.Errorf("wasm.interface_method_signature")
		}
	}
	methodGraph, err := normalizePureLocals(graph, body.Reference, map[wire.ID]string{parameter.ID: "i64"})
	if err != nil {
		return nil, err
	}
	block := methodGraph.Entities[body.Reference]
	statements, err := field(block, 0x9800)
	if err != nil || statements.Tag != 7 || len(statements.List) != 1 || statements.List[0].Tag != 6 {
		return nil, fmt.Errorf("wasm.interface_method_body")
	}
	ret := methodGraph.Entities[statements.List[0].Reference]
	values, err := field(ret, 0x9810)
	if ret.Schema != identity(0x9081) || err != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
		return nil, fmt.Errorf("wasm.interface_method_return")
	}
	return lowerTransitionI64(methodGraph, values.List[0].Reference, map[wire.ID]byte{parameter.ID: 1}, &receiverBinding.ID, map[wire.ID]bool{})
}

func pureInterfaceModule(cases []interfaceDispatchCase, abi PureABI) ([]byte, PureABI, error) {
	var wasm bytes.Buffer
	wasm.Write([]byte{'\x00', 'a', 's', 'm', '\x01', 0, 0, 0})
	var types bytes.Buffer
	uleb(&types, 6)
	functionType(&types, []byte{0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, nil, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7e, 0x7e}, []byte{0x7e})
	functionType(&types, []byte{0x7f, 0x7f}, nil)
	section(&wasm, 1, types.Bytes())
	var functions bytes.Buffer
	uleb(&functions, uint64(6+len(cases)))
	for _, typeIndex := range []byte{0, 5, 3, 3, 2} {
		uleb(&functions, uint64(typeIndex))
	}
	for range cases {
		uleb(&functions, 4)
	}
	uleb(&functions, 1)
	section(&wasm, 3, functions.Bytes())
	section(&wasm, 5, []byte{1, 0, 1})
	var globals bytes.Buffer
	globals.Write([]byte{1, 0x7f, 1, 0x41})
	sleb(&globals, 1024)
	globals.WriteByte(0x0b)
	section(&wasm, 6, globals.Bytes())
	providerIndex := uint64(5 + len(cases))
	var exports bytes.Buffer
	uleb(&exports, 7)
	export(&exports, "memory", 2, 0)
	export(&exports, "pulp_alloc", 0, 0)
	export(&exports, "pulp_free", 0, 1)
	export(&exports, "pulp_init", 0, 2)
	export(&exports, "pulp_step", 0, 3)
	export(&exports, "pulp_shutdown", 0, 4)
	export(&exports, "pulp_on_call", 0, providerIndex)
	section(&wasm, 7, exports.Bytes())
	helper := func(code []byte) []byte { return append(append([]byte{0}, code...), 0x0b) }
	bodies := [][]byte{pureAllocatorBody(), pureFreeBody(), {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}}
	for _, dispatch := range cases {
		bodies = append(bodies, helper(dispatch.code))
	}
	var provider bytes.Buffer
	provider.Write([]byte{1, 1, 0x7e})
	provider.Write([]byte{0x20, 3, 0x41, 0x18, 0x47, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b})
	for index := range cases {
		provider.Write([]byte{0x20, 2, 0x29, 3, 0})
		provider.WriteByte(0x42)
		sleb(&provider, int64(index+1))
		provider.Write([]byte{0x51, 0x04, 0x40})
		provider.Write([]byte{0x20, 2, 0x29, 3, 8, 0x20, 2, 0x29, 3, 16, 0x10})
		uleb(&provider, uint64(5+index))
		provider.Write([]byte{0x21, 6})
		constI32(&provider, 8192)
		provider.Write([]byte{0x20, 6, 0x37, 3, 0})
		provider.Write([]byte{0x20, 4})
		constI32(&provider, 8192)
		provider.Write([]byte{0x36, 2, 0, 0x20, 5, 0x41, 8, 0x36, 2, 0, 0x41, 0, 0x0f, 0x0b})
	}
	provider.WriteByte(0x00)
	provider.WriteByte(0x0b)
	bodies = append(bodies, provider.Bytes())
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, body := range bodies {
		uleb(&code, uint64(len(body)))
		code.Write(body)
	}
	section(&wasm, 10, code.Bytes())
	return wasm.Bytes(), abi, nil
}

func pureInterfaceDispatchModule(cases []interfaceDispatchCase, abi PureABI) ([]byte, PureABI, error) {
	if len(cases) != 2 {
		return nil, PureABI{}, fmt.Errorf("wasm.interface_dispatch_cases")
	}
	var wasm bytes.Buffer
	wasm.Write([]byte{'\x00', 'a', 's', 'm', '\x01', 0, 0, 0})
	var types bytes.Buffer
	uleb(&types, 6)
	functionType(&types, []byte{0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, nil, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7e, 0x7e}, []byte{0x7e})
	functionType(&types, []byte{0x7f, 0x7f}, nil)
	section(&wasm, 1, types.Bytes())
	section(&wasm, 3, []byte{8, 0, 5, 3, 3, 2, 4, 4, 1})
	section(&wasm, 5, []byte{1, 0, 1})
	var globals bytes.Buffer
	globals.Write([]byte{1, 0x7f, 1, 0x41})
	sleb(&globals, 1024)
	globals.WriteByte(0x0b)
	section(&wasm, 6, globals.Bytes())
	var exports bytes.Buffer
	uleb(&exports, 7)
	export(&exports, "memory", 2, 0)
	export(&exports, "pulp_alloc", 0, 0)
	export(&exports, "pulp_free", 0, 1)
	export(&exports, "pulp_init", 0, 2)
	export(&exports, "pulp_step", 0, 3)
	export(&exports, "pulp_shutdown", 0, 4)
	export(&exports, "pulp_on_call", 0, 7)
	section(&wasm, 7, exports.Bytes())
	helper := func(code []byte) []byte { return append(append([]byte{0}, code...), 0x0b) }
	bodies := [][]byte{pureAllocatorBody(), pureFreeBody(), {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, helper(cases[0].code), helper(cases[1].code)}
	var provider bytes.Buffer
	provider.Write([]byte{1, 1, 0x7e})
	provider.Write([]byte{0x20, 3, 0x41, 0x11, 0x47, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b})
	// Reject non-canonical Boolean encodings before selecting the witness.
	provider.Write([]byte{0x20, 2, 0x2d, 0, 0, 0x41, 1, 0x4b, 0x04, 0x40, 0x00, 0x0b})
	provider.Write([]byte{0x20, 2, 0x2d, 0, 0, 0x04, 0x7e, 0x20, 2, 0x29, 3, 1, 0x20, 2, 0x29, 3, 9, 0x10, 6, 0x05, 0x20, 2, 0x29, 3, 1, 0x20, 2, 0x29, 3, 9, 0x10, 5, 0x0b, 0x21, 6})
	constI32(&provider, 8192)
	provider.Write([]byte{0x20, 6, 0x37, 3, 0})
	provider.Write([]byte{0x20, 4})
	constI32(&provider, 8192)
	provider.Write([]byte{0x36, 2, 0, 0x20, 5, 0x41, 8, 0x36, 2, 0, 0x41, 0, 0x0b})
	bodies = append(bodies, provider.Bytes())
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, body := range bodies {
		uleb(&code, uint64(len(body)))
		code.Write(body)
	}
	section(&wasm, 10, code.Bytes())
	return wasm.Bytes(), abi, nil
}
