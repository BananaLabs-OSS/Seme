package goprovider

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
)

type goExpressionKind uint8

const (
	goParameterRead goExpressionKind = iota + 1
	goIntegerAdd
	goIntegerLessEqual
	goIntegerLiteral
	goIntegerMultiply
	goIntegerSubtract
	goBooleanLiteral
	goBooleanAnd
	goStringLiteral
	goBooleanOr
	goStringEqual
	goStringConcat
	goLocalRead
	goFunctionCall
	goRecordConstruct
	goFieldRead
	goPlaceRead
	goFixedArrayConstruct
	goIndexRead
	goIterationBindingRead
	goFold
	goCollectionLength
	goDynamicIndexRead
	goCollectionAppend
	goCollectionUpdate
	goReceiverRead
	goMethodCall
	goStateTransition
	goTransitionState
	goTransitionResult
	goDynamicMethodCall
	goInterfaceValue
	goCaptureRead
	goClosureParameterRead
	goClosureConstruct
	goIndirectCall
	goMutableCaptureRead
	goCaptureUpdate
	goSequence
	goMutableClosureConstruct
	goStatefulIndirectCall
	goEmptyMap
	goMapLookup
	goMapUpdate
	goBytesLiteral
	goOptionNone
	goOptionSome
	goResultOk
	goResultError
	goVariantRead
	goOptionMatch
	goResultMatch
)

// goExpression is the provider's small typed source-expression tree. It keeps
// Go AST details out of canonical emission and normalizes equivalent source
// spellings before a target profile decides which tree shapes it supports.
type goExpression struct {
	kind        goExpressionKind
	parameter   int
	local       int
	integer     uint64
	boolean     bool
	text        string
	left        *goExpression
	right       *goExpression
	callee      string
	arguments   []*goExpression
	recordType  string
	field       string
	values      []*goExpression
	mutable     bool
	arrayType   string
	arrayLen    uint64
	initial     *goExpression
	body        *goExpression
	bindingID   string
	accName     string
	elementName string
	receiverID  string
	methodID    string
	typeID      string
	witnessID   string
	alternate   *goExpression
	errorName   string
	errorTypeID string
}

func emitCanonicalExpression(expression *goExpression, owner string, parameterIDs []string, integerID string) ([]graphEntity, string, error) {
	return emitCanonicalExpressionWithLocals(expression, owner, parameterIDs, nil, integerID)
}

func emitCanonicalExpressionWithLocals(expression *goExpression, owner string, parameterIDs []string, localIDs map[int]string, integerID string) ([]graphEntity, string, error) {
	emitted := make(map[string]graphEntity)
	var emit func(*goExpression, string) (string, error)
	emit = func(expression *goExpression, path string) (string, error) {
		if expression == nil {
			return "", fmt.Errorf("expression.nil")
		}
		switch expression.kind {
		case goParameterRead:
			if expression.parameter < 0 || expression.parameter >= len(parameterIDs) {
				return "", fmt.Errorf("expression.parameter_out_of_range")
			}
			id := stableID("execution", owner, "read", strconv.Itoa(expression.parameter))
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009013", []graphField{refField(0x9130, parameterIDs[expression.parameter])})}
			return id, nil
		case goLocalRead:
			bindingID, ok := localIDs[expression.local]
			if !ok {
				return "", fmt.Errorf("expression.local_out_of_scope")
			}
			id := stableID("execution", owner, "local-read", bindingID)
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090d2", []graphField{refField(0x9d20, bindingID)})}
			return id, nil
		case goPlaceRead:
			placeID, ok := localIDs[expression.local]
			if !ok {
				return "", fmt.Errorf("expression.place_out_of_scope")
			}
			id := stableID("execution", owner, "place-read", placeID)
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090e2", []graphField{refField(0x9e20, placeID)})}
			return id, nil
		case goFunctionCall:
			if expression.callee == "" {
				return "", fmt.Errorf("expression.call_missing_callee")
			}
			arguments := make([]string, len(expression.arguments))
			for index, argument := range expression.arguments {
				id, err := emit(argument, path+".argument."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
				arguments[index] = id
			}
			id := expressionNodeID(owner, path, "function-call")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009060", []graphField{refField(0x9600, expression.callee), refsField(0x9601, arguments)})}
			return id, nil
		case goRecordConstruct:
			values := make([]string, len(expression.values))
			for index, value := range expression.values {
				id, err := emit(value, path+".field."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
				values[index] = id
			}
			id := expressionNodeID(owner, path, "record-construct")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009033", []graphField{refField(0x9330, expression.recordType), refsField(0x9331, values)})}
			return id, nil
		case goFieldRead:
			record, err := emit(expression.left, path+".record")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "field-read")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009032", []graphField{refField(0x9320, record), refField(0x9321, expression.field)})}
			return id, nil
		case goFixedArrayConstruct:
			values := make([]string, len(expression.values))
			for index, value := range expression.values {
				id, err := emit(value, path+".element."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
				values[index] = id
			}
			emitted[expression.arrayType] = graphEntity{expression.arrayType, entity(expression.arrayType, "000000000000000000000000000090f2", []graphField{refField(0x9f20, integerID), unsignedField(0x9f21, expression.arrayLen)})}
			id := expressionNodeID(owner, path, "fixed-array-construct")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090f3", []graphField{refField(0x9f30, expression.arrayType), refsField(0x9f31, values)})}
			return id, nil
		case goIndexRead:
			collection, err := emit(expression.left, path+".collection")
			if err != nil {
				return "", err
			}
			index, err := emit(expression.right, path+".index")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "index-read")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090f4", []graphField{refField(0x9f40, collection), refField(0x9f41, index)})}
			return id, nil
		case goIterationBindingRead:
			if expression.bindingID == "" {
				return "", fmt.Errorf("expression.iteration_binding_missing")
			}
			id := expressionNodeID(owner, path, "iteration-binding-read")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090f6", []graphField{refField(0x9f60, expression.bindingID)})}
			return id, nil
		case goFold:
			collection, err := emit(expression.left, path+".collection")
			if err != nil {
				return "", err
			}
			initial, err := emit(expression.initial, path+".initial")
			if err != nil {
				return "", err
			}
			accumulatorID := expressionNodeID(owner, path, "fold-accumulator-binding")
			elementID := expressionNodeID(owner, path, "fold-element-binding")
			assignFoldBinding(expression.body, "accumulator", accumulatorID)
			assignFoldBinding(expression.body, "element", elementID)
			body, err := emit(expression.body, path+".body")
			if err != nil {
				return "", err
			}
			accumulatorType := integerID
			if expression.typeID != "" {
				accumulatorType = expression.typeID
			}
			emitted[accumulatorID] = graphEntity{accumulatorID, entity(accumulatorID, "000000000000000000000000000090f5", []graphField{bytesField(0x9f50, expression.accName), refField(0x9f51, accumulatorType)})}
			emitted[elementID] = graphEntity{elementID, entity(elementID, "000000000000000000000000000090f5", []graphField{bytesField(0x9f50, expression.elementName), refField(0x9f51, integerID)})}
			id := expressionNodeID(owner, path, "fold")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090f7", []graphField{refField(0x9f70, collection), refField(0x9f71, initial), refField(0x9f72, accumulatorID), refField(0x9f73, elementID), refField(0x9f74, body)})}
			return id, nil
		case goCollectionLength:
			collection, err := emit(expression.left, path+".collection")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "collection-length")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090f9", []graphField{refField(0x9f90, collection)})}
			return id, nil
		case goDynamicIndexRead:
			collection, err := emit(expression.left, path+".collection")
			if err != nil {
				return "", err
			}
			index, err := emit(expression.right, path+".index")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "dynamic-index-read")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090fa", []graphField{refField(0x9fa0, collection), refField(0x9fa1, index)})}
			return id, nil
		case goCollectionAppend:
			collection, err := emit(expression.left, path+".collection")
			if err != nil {
				return "", err
			}
			value, err := emit(expression.right, path+".value")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "collection-append")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090fb", []graphField{refField(0x9fb0, collection), refField(0x9fb1, value)})}
			return id, nil
		case goCollectionUpdate:
			collection, err := emit(expression.left, path+".collection")
			if err != nil {
				return "", err
			}
			index, err := emit(expression.initial, path+".index")
			if err != nil {
				return "", err
			}
			value, err := emit(expression.right, path+".value")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "collection-update")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090fc", []graphField{refField(0x9fc0, collection), refField(0x9fc1, index), refField(0x9fc2, value)})}
			return id, nil
		case goReceiverRead:
			if expression.receiverID == "" {
				return "", fmt.Errorf("expression.receiver_missing")
			}
			id := expressionNodeID(owner, path, "receiver-read")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a001", []graphField{refField(0xa0010, expression.receiverID)})}
			return id, nil
		case goMethodCall:
			receiver, err := emit(expression.left, path+".receiver")
			if err != nil {
				return "", err
			}
			arguments := make([]string, len(expression.arguments))
			for index, argument := range expression.arguments {
				arguments[index], err = emit(argument, path+".argument."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
			}
			id := expressionNodeID(owner, path, "method-call")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a003", []graphField{refField(0xa0030, receiver), refField(0xa0031, expression.methodID), refsField(0xa0032, arguments)})}
			return id, nil
		case goDynamicMethodCall:
			receiver, err := emit(expression.left, path+".receiver")
			if err != nil {
				return "", err
			}
			arguments := make([]string, len(expression.arguments))
			for index, argument := range expression.arguments {
				arguments[index], err = emit(argument, path+".argument."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
			}
			id := expressionNodeID(owner, path, "dynamic-method-call")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a014", []graphField{refField(0xa0140, receiver), refField(0xa0141, expression.methodID), refsField(0xa0142, arguments)})}
			return id, nil
		case goInterfaceValue:
			value, err := emit(expression.left, path+".value")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "interface-value")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a013", []graphField{refField(0xa0130, expression.typeID), refField(0xa0131, value), refField(0xa0132, expression.witnessID)})}
			return id, nil
		case goCaptureRead:
			id := expressionNodeID(owner, path, "capture-read")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a022", []graphField{refField(0xa0220, expression.bindingID)})}
			return id, nil
		case goClosureParameterRead:
			id := expressionNodeID(owner, path, "closure-parameter-read")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009013", []graphField{refField(0x9130, expression.bindingID)})}
			return id, nil
		case goClosureConstruct:
			captured, err := emit(expression.left, path+".capture.value")
			if err != nil {
				return "", err
			}
			captureID := expressionNodeID(owner, path, "capture")
			parameterID := expressionNodeID(owner, path, "closure-parameter")
			assignClosureBindings(expression.body, captureID, parameterID)
			body, err := emit(expression.body, path+".body")
			if err != nil {
				return "", err
			}
			emitted[expression.typeID] = graphEntity{expression.typeID, entity(expression.typeID, "0000000000000000000000000000a020", []graphField{refsField(0xa0200, []string{integerID}), refField(0xa0201, integerID)})}
			emitted[captureID] = graphEntity{captureID, entity(captureID, "0000000000000000000000000000a021", []graphField{bytesField(0xa0210, expression.text), refField(0xa0211, integerID), refField(0xa0212, captured)})}
			emitted[parameterID] = graphEntity{parameterID, entity(parameterID, "00000000000000000000000000009012", []graphField{bytesField(0x9120, expression.elementName), refField(0x9121, integerID), unsignedField(0x9122, 0)})}
			id := expressionNodeID(owner, path, "closure")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a023", []graphField{refField(0xa0230, expression.typeID), refsField(0xa0231, []string{parameterID}), refsField(0xa0232, []string{captureID}), refField(0xa0233, body)})}
			return id, nil
		case goIndirectCall:
			callee, err := emit(expression.left, path+".callee")
			if err != nil {
				return "", err
			}
			arguments := make([]string, len(expression.arguments))
			for index, argument := range expression.arguments {
				arguments[index], err = emit(argument, path+".argument."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
			}
			id := expressionNodeID(owner, path, "indirect-call")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a024", []graphField{refField(0xa0240, callee), refsField(0xa0241, arguments)})}
			return id, nil
		case goMutableCaptureRead:
			id := expressionNodeID(owner, path, "mutable-capture-read")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a031", []graphField{refField(0xa0310, expression.bindingID)})}
			return id, nil
		case goCaptureUpdate:
			value, err := emit(expression.left, path+".value")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "capture-update")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a032", []graphField{refField(0xa0320, expression.bindingID), refField(0xa0321, value)})}
			return id, nil
		case goSequence:
			steps := make([]string, len(expression.arguments))
			var err error
			for index, step := range expression.arguments {
				steps[index], err = emit(step, path+".step."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
			}
			result, err := emit(expression.left, path+".result")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "sequence")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a033", []graphField{refsField(0xa0330, steps), refField(0xa0331, result)})}
			return id, nil
		case goMutableClosureConstruct:
			initial, err := emit(expression.left, path+".capture.initial")
			if err != nil {
				return "", err
			}
			captureID := expressionNodeID(owner, path, "mutable-capture")
			parameterID := expressionNodeID(owner, path, "closure-parameter")
			assignMutableClosureBindings(expression.body, captureID, parameterID)
			body, err := emit(expression.body, path+".body")
			if err != nil {
				return "", err
			}
			emitted[expression.typeID] = graphEntity{expression.typeID, entity(expression.typeID, "0000000000000000000000000000a020", []graphField{refsField(0xa0200, []string{integerID}), refField(0xa0201, integerID)})}
			emitted[captureID] = graphEntity{captureID, entity(captureID, "0000000000000000000000000000a030", []graphField{bytesField(0xa0300, expression.text), refField(0xa0301, integerID), refField(0xa0302, initial)})}
			emitted[parameterID] = graphEntity{parameterID, entity(parameterID, "00000000000000000000000000009012", []graphField{bytesField(0x9120, expression.elementName), refField(0x9121, integerID), unsignedField(0x9122, 0)})}
			id := expressionNodeID(owner, path, "mutable-closure")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a034", []graphField{refField(0xa0340, expression.typeID), refsField(0xa0341, []string{parameterID}), refsField(0xa0342, []string{captureID}), refField(0xa0343, body)})}
			return id, nil
		case goStatefulIndirectCall:
			callee, err := emit(expression.left, path+".callee")
			if err != nil {
				return "", err
			}
			arguments := make([]string, len(expression.arguments))
			for index, argument := range expression.arguments {
				arguments[index], err = emit(argument, path+".argument."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
			}
			id := expressionNodeID(owner, path, "stateful-indirect-call")
			emitted[expression.typeID] = graphEntity{expression.typeID, entity(expression.typeID, "0000000000000000000000000000a004", []graphField{refField(0xa0040, goUnaryI64FunctionTypeID()), refField(0xa0041, integerID)})}
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a035", []graphField{refField(0xa0350, callee), refsField(0xa0351, arguments)})}
			return id, nil
		case goEmptyMap:
			emitted[expression.typeID] = graphEntity{expression.typeID, entity(expression.typeID, "0000000000000000000000000000a040", []graphField{refField(0xa0400, integerID), refField(0xa0401, integerID)})}
			id := expressionNodeID(owner, path, "empty-map")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a041", []graphField{refField(0xa0410, expression.typeID)})}
			return id, nil
		case goMapLookup:
			mapping, err := emit(expression.left, path+".map")
			if err != nil {
				return "", err
			}
			key, err := emit(expression.right, path+".key")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "map-lookup")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a042", []graphField{refField(0xa0420, mapping), refField(0xa0421, key)})}
			return id, nil
		case goMapUpdate:
			mapping, err := emit(expression.left, path+".map")
			if err != nil {
				return "", err
			}
			key, err := emit(expression.initial, path+".key")
			if err != nil {
				return "", err
			}
			value, err := emit(expression.right, path+".value")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "map-update")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a043", []graphField{refField(0xa0430, mapping), refField(0xa0431, key), refField(0xa0432, value)})}
			return id, nil
		case goBytesLiteral:
			id := expressionNodeID(owner, path, "bytes-literal")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a064", []graphField{bytesHexField(0xa0640, expression.text)})}
			return id, nil
		case goOptionNone:
			id := expressionNodeID(owner, path, "option-none")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a051", []graphField{refField(0xa0510, expression.typeID)})}
			return id, nil
		case goOptionSome, goResultOk, goResultError:
			value, err := emit(expression.left, path+".value")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, map[goExpressionKind]string{goOptionSome: "option-some", goResultOk: "result-ok", goResultError: "result-error"}[expression.kind])
			schema, typeField, valueField := "0000000000000000000000000000a052", uint64(0xa0520), uint64(0xa0521)
			if expression.kind == goResultOk {
				schema, typeField, valueField = "00000000000000000000000000009043", 0x9410, 0x9411
			} else if expression.kind == goResultError {
				schema, typeField, valueField = "00000000000000000000000000009044", 0x9420, 0x9421
			}
			emitted[id] = graphEntity{id, entity(id, schema, []graphField{refField(typeField, expression.typeID), refField(valueField, value)})}
			return id, nil
		case goVariantRead:
			id := expressionNodeID(owner, path, "variant-read")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a061", []graphField{refField(0xa0610, expression.bindingID)})}
			return id, nil
		case goOptionMatch:
			if expression.bindingID == "" {
				expression.bindingID = stableID("execution", owner, path, "some-binding")
			}
			bindVariantReads(expression.body, expression.bindingID)
			value, err := emit(expression.left, path+".value")
			if err != nil {
				return "", err
			}
			noneValue, err := emit(expression.initial, path+".none.value")
			if err != nil {
				return "", err
			}
			someValue, err := emit(expression.body, path+".some.value")
			if err != nil {
				return "", err
			}
			noneReturn, noneBlock := expressionNodeID(owner, path, "option-none-return"), expressionNodeID(owner, path, "option-none-block")
			someReturn, someBlock := expressionNodeID(owner, path, "option-some-return"), expressionNodeID(owner, path, "option-some-block")
			emitted[expression.bindingID] = graphEntity{expression.bindingID, entity(expression.bindingID, "0000000000000000000000000000a060", []graphField{bytesField(0xa0600, expression.text), refField(0xa0601, expression.typeID)})}
			emitted[noneReturn] = graphEntity{noneReturn, entity(noneReturn, "00000000000000000000000000009081", []graphField{refsField(0x9810, []string{noneValue})})}
			emitted[noneBlock] = graphEntity{noneBlock, entity(noneBlock, "00000000000000000000000000009080", []graphField{refsField(0x9800, []string{noneReturn})})}
			emitted[someReturn] = graphEntity{someReturn, entity(someReturn, "00000000000000000000000000009081", []graphField{refsField(0x9810, []string{someValue})})}
			emitted[someBlock] = graphEntity{someBlock, entity(someBlock, "00000000000000000000000000009080", []graphField{refsField(0x9800, []string{someReturn})})}
			id := expressionNodeID(owner, path, "option-match")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a063", []graphField{refField(0xa0630, value), refField(0xa0631, noneBlock), refField(0xa0632, expression.bindingID), refField(0xa0633, someBlock)})}
			return id, nil
		case goResultMatch:
			if expression.bindingID == "" {
				expression.bindingID = stableID("execution", owner, path, "ok-binding")
			}
			if expression.witnessID == "" {
				expression.witnessID = stableID("execution", owner, path, "error-binding")
			}
			bindVariantReads(expression.body, expression.bindingID)
			bindVariantReads(expression.alternate, expression.witnessID)
			value, err := emit(expression.left, path+".value")
			if err != nil {
				return "", err
			}
			okValue, err := emit(expression.body, path+".ok.value")
			if err != nil {
				return "", err
			}
			errorValue, err := emit(expression.alternate, path+".error.value")
			if err != nil {
				return "", err
			}
			okReturn, okBlock := expressionNodeID(owner, path, "result-ok-return"), expressionNodeID(owner, path, "result-ok-block")
			errorReturn, errorBlock := expressionNodeID(owner, path, "result-error-return"), expressionNodeID(owner, path, "result-error-block")
			emitted[expression.bindingID] = graphEntity{expression.bindingID, entity(expression.bindingID, "0000000000000000000000000000a060", []graphField{bytesField(0xa0600, expression.text), refField(0xa0601, expression.typeID)})}
			emitted[expression.witnessID] = graphEntity{expression.witnessID, entity(expression.witnessID, "0000000000000000000000000000a060", []graphField{bytesField(0xa0600, expression.errorName), refField(0xa0601, expression.errorTypeID)})}
			emitted[okReturn] = graphEntity{okReturn, entity(okReturn, "00000000000000000000000000009081", []graphField{refsField(0x9810, []string{okValue})})}
			emitted[okBlock] = graphEntity{okBlock, entity(okBlock, "00000000000000000000000000009080", []graphField{refsField(0x9800, []string{okReturn})})}
			emitted[errorReturn] = graphEntity{errorReturn, entity(errorReturn, "00000000000000000000000000009081", []graphField{refsField(0x9810, []string{errorValue})})}
			emitted[errorBlock] = graphEntity{errorBlock, entity(errorBlock, "00000000000000000000000000009080", []graphField{refsField(0x9800, []string{errorReturn})})}
			id := expressionNodeID(owner, path, "result-match")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a062", []graphField{refField(0xa0620, value), refField(0xa0621, expression.bindingID), refField(0xa0622, okBlock), refField(0xa0623, expression.witnessID), refField(0xa0624, errorBlock)})}
			return id, nil
		case goStateTransition:
			state, err := emit(expression.left, path+".state")
			if err != nil {
				return "", err
			}
			result, err := emit(expression.right, path+".result")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "state-transition")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a005", []graphField{refField(0xa0050, expression.typeID), refField(0xa0051, state), refField(0xa0052, result)})}
			return id, nil
		case goTransitionState, goTransitionResult:
			value, err := emit(expression.left, path+".value")
			if err != nil {
				return "", err
			}
			kind, schema, field := "transition-state", "0000000000000000000000000000a006", uint64(0xa0060)
			if expression.kind == goTransitionResult {
				kind, schema, field = "transition-result", "0000000000000000000000000000a007", 0xa0070
			}
			id := expressionNodeID(owner, path, kind)
			emitted[id] = graphEntity{id, entity(id, schema, []graphField{refField(field, value)})}
			return id, nil
		case goIntegerLiteral:
			id := expressionNodeID(owner, path, "integer-literal")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009070", []graphField{unsignedField(0x9700, expression.integer), refField(0x9701, integerID)})}
			return id, nil
		case goBooleanLiteral:
			id := expressionNodeID(owner, path, "boolean-literal")
			value := "fa"
			if expression.boolean {
				value = "tr"
			}
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090b0", []graphField{{0x9b00, value}})}
			return id, nil
		case goBooleanAnd:
			left, err := emit(expression.left, path+".left")
			if err != nil {
				return "", err
			}
			right, err := emit(expression.right, path+".right")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "boolean-and")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090b1", []graphField{refField(0x9b10, left), refField(0x9b11, right)})}
			return id, nil
		case goBooleanOr, goStringEqual, goStringConcat:
			left, err := emit(expression.left, path+".left")
			if err != nil {
				return "", err
			}
			right, err := emit(expression.right, path+".right")
			if err != nil {
				return "", err
			}
			kind, schema, leftField, rightField := "boolean-or", "000000000000000000000000000090c1", uint64(0x9c10), uint64(0x9c11)
			if expression.kind == goStringEqual {
				kind, schema, leftField, rightField = "string-equal", "000000000000000000000000000090c2", 0x9c20, 0x9c21
			}
			if expression.kind == goStringConcat {
				kind, schema, leftField, rightField = "string-concat", "000000000000000000000000000090c3", 0x9c30, 0x9c31
			}
			id := expressionNodeID(owner, path, kind)
			emitted[id] = graphEntity{id, entity(id, schema, []graphField{refField(leftField, left), refField(rightField, right)})}
			return id, nil
		case goStringLiteral:
			id := expressionNodeID(owner, path, "string-literal")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009050", []graphField{bytesField(0x9500, expression.text)})}
			return id, nil
		case goIntegerAdd, goIntegerMultiply, goIntegerSubtract, goIntegerLessEqual:
			left, err := emit(expression.left, path+".left")
			if err != nil {
				return "", err
			}
			right, err := emit(expression.right, path+".right")
			if err != nil {
				return "", err
			}
			if expression.kind == goIntegerAdd {
				id := expressionNodeID(owner, path, "add")
				emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009014", []graphField{refField(0x9140, left), refField(0x9141, right), refField(0x9142, integerID)})}
				return id, nil
			}
			if expression.kind == goIntegerMultiply {
				id := expressionNodeID(owner, path, "multiply")
				emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009090", []graphField{refField(0x9900, left), refField(0x9901, right), refField(0x9902, integerID)})}
				return id, nil
			}
			if expression.kind == goIntegerSubtract {
				id := expressionNodeID(owner, path, "subtract")
				emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090a0", []graphField{refField(0x9a00, left), refField(0x9a01, right), refField(0x9a02, integerID)})}
				return id, nil
			}
			id := expressionNodeID(owner, path, "less-equal")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009021", []graphField{refField(0x9160, left), refField(0x9161, right), refField(0x9162, integerID)})}
			return id, nil
		default:
			return "", fmt.Errorf("expression.unsupported_kind")
		}
	}
	root, err := emit(expression, "root")
	if err != nil {
		return nil, "", err
	}
	entities := make([]graphEntity, 0, len(emitted))
	for _, item := range emitted {
		entities = append(entities, item)
	}
	return entities, root, nil
}

func keyedCompositeFields(expression *ast.CompositeLit) (map[string]ast.Expr, error) {
	fields := make(map[string]ast.Expr, len(expression.Elts))
	for _, raw := range expression.Elts {
		keyed, ok := raw.(*ast.KeyValueExpr)
		if !ok {
			return nil, fmt.Errorf("expression.tagged_constructor_requires_keys")
		}
		name, ok := keyed.Key.(*ast.Ident)
		if !ok || fields[name.Name] != nil {
			return nil, fmt.Errorf("expression.tagged_constructor_field")
		}
		fields[name.Name] = keyed.Value
	}
	return fields, nil
}
func bindVariantReads(expression *goExpression, binding string) {
	if expression == nil {
		return
	}
	if expression.kind == goVariantRead {
		expression.bindingID = binding
	}
	bindVariantReads(expression.left, binding)
	bindVariantReads(expression.right, binding)
	bindVariantReads(expression.initial, binding)
	bindVariantReads(expression.body, binding)
	for _, item := range expression.arguments {
		bindVariantReads(item, binding)
	}
	for _, item := range expression.values {
		bindVariantReads(item, binding)
	}
}

func goUnaryI64FunctionTypeID() string {
	return stableID("execution", "type", "function", "i64", "i64")
}

func goStatefulUnaryI64TransitionTypeID() string {
	return stableID("execution", "type", "state-transition", goUnaryI64FunctionTypeID(), stableID("execution", "type", "i64"))
}

func assignFoldBinding(expression *goExpression, role, id string) {
	if expression == nil {
		return
	}
	if expression.kind == goIterationBindingRead && expression.text == role {
		expression.bindingID = id
	}
	assignFoldBinding(expression.left, role, id)
	assignFoldBinding(expression.right, role, id)
	assignFoldBinding(expression.initial, role, id)
	assignFoldBinding(expression.body, role, id)
	for _, child := range expression.arguments {
		assignFoldBinding(child, role, id)
	}
}

func assignClosureBindings(expression *goExpression, captureID, parameterID string) {
	if expression == nil {
		return
	}
	if expression.kind == goCaptureRead {
		expression.bindingID = captureID
	}
	if expression.kind == goClosureParameterRead {
		expression.bindingID = parameterID
	}
	assignClosureBindings(expression.left, captureID, parameterID)
	assignClosureBindings(expression.right, captureID, parameterID)
}

func assignMutableClosureBindings(expression *goExpression, captureID, parameterID string) {
	if expression == nil {
		return
	}
	if expression.kind == goMutableCaptureRead || expression.kind == goCaptureUpdate {
		expression.bindingID = captureID
	}
	if expression.kind == goClosureParameterRead {
		expression.bindingID = parameterID
	}
	assignMutableClosureBindings(expression.left, captureID, parameterID)
	assignMutableClosureBindings(expression.right, captureID, parameterID)
	for _, child := range expression.arguments {
		assignMutableClosureBindings(child, captureID, parameterID)
	}
}

func expressionNodeID(owner, path, kind string) string {
	// Preserve the frozen v6 identities for the original decision shape. New
	// nested nodes use their normalized semantic path and do not renumber peers.
	if path == "root" && kind == "less-equal" {
		return stableID("execution", owner, "less-equal")
	}
	if path == "root.left" && kind == "add" {
		return stableID("execution", owner, "add")
	}
	return stableID("execution", owner, "expression", path, kind)
}

func analyzeGoExpression(expression ast.Expr, signature *types.Signature, info *types.Info) (*goExpression, error) {
	return analyzeGoExpressionWithLocals(expression, signature, info, nil)
}

func analyzeGoExpressionWithLocals(expression ast.Expr, signature *types.Signature, info *types.Info, locals map[types.Object]int) (*goExpression, error) {
	return analyzeGoExpressionWithContext(expression, signature, info, locals, nil)
}

func analyzeGoExpressionWithContext(expression ast.Expr, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string) (*goExpression, error) {
	return analyzeGoExpressionWithProgram(expression, signature, info, locals, functions, nil, nil)
}

type goRecordInfo struct {
	id      string
	fields  map[*types.Var]string
	ordered []*types.Var
}

func analyzeGoExpressionWithProgram(expression ast.Expr, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutableLocals map[types.Object]bool) (*goExpression, error) {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		if signature.Recv() != nil && info.Uses[expression] == signature.Recv() {
			return &goExpression{kind: goReceiverRead, receiverID: goReceiverID(signature)}, nil
		}
		for index := 0; index < signature.Params().Len(); index++ {
			if info.Uses[expression] == signature.Params().At(index) {
				return &goExpression{kind: goParameterRead, parameter: index}, nil
			}
		}
		if local, ok := locals[info.Uses[expression]]; ok {
			kind := goLocalRead
			if mutableLocals[info.Uses[expression]] {
				kind = goPlaceRead
			}
			return &goExpression{kind: kind, local: local}, nil
		}
		if object, ok := info.Uses[expression].(*types.Const); ok && object.Type() == types.Typ[types.UntypedBool] {
			return &goExpression{kind: goBooleanLiteral, boolean: constant.BoolVal(object.Val())}, nil
		}
		return nil, fmt.Errorf("expression.unresolved_parameter")
	case *ast.BasicLit:
		if expression.Kind == token.STRING {
			value, err := strconv.Unquote(expression.Value)
			if err != nil {
				return nil, fmt.Errorf("expression.invalid_string_literal")
			}
			return &goExpression{kind: goStringLiteral, text: value}, nil
		}
		if expression.Kind != token.INT {
			return nil, fmt.Errorf("expression.unsupported_literal")
		}
		value, err := strconv.ParseInt(expression.Value, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("expression.invalid_integer_literal")
		}
		return &goExpression{kind: goIntegerLiteral, integer: uint64(value)}, nil
	case *ast.BinaryExpr:
		left, right := expression.X, expression.Y
		kind := goExpressionKind(0)
		switch expression.Op {
		case token.ADD:
			if isGoStringExpression(expression, info) {
				kind = goStringConcat
			} else {
				kind = goIntegerAdd
			}
		case token.MUL:
			kind = goIntegerMultiply
		case token.SUB:
			kind = goIntegerSubtract
		case token.LAND:
			kind = goBooleanAnd
		case token.LOR:
			kind = goBooleanOr
		case token.EQL:
			if isGoStringExpression(expression.X, info) {
				kind = goStringEqual
			} else {
				return nil, fmt.Errorf("expression.unsupported_operator:%s", expression.Op)
			}
		case token.LEQ:
			kind = goIntegerLessEqual
		case token.GEQ:
			kind = goIntegerLessEqual
			left, right = right, left
		default:
			return nil, fmt.Errorf("expression.unsupported_operator:%s", expression.Op)
		}
		analyzedLeft, err := analyzeGoExpressionWithProgram(left, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		analyzedRight, err := analyzeGoExpressionWithProgram(right, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		return &goExpression{kind: kind, left: analyzedLeft, right: analyzedRight}, nil
	case *ast.CallExpr:
		if selector, ok := ast.Unparen(expression.Fun).(*ast.SelectorExpr); ok {
			if selection := info.Selections[selector]; selection != nil && selection.Kind() == types.MethodVal {
				methodID, exists := functions[selection.Obj()]
				if !exists || expression.Ellipsis.IsValid() {
					return nil, fmt.Errorf("expression.unsupported_method_call")
				}
				receiver, err := analyzeGoExpressionWithProgram(selector.X, signature, info, locals, functions, records, mutableLocals)
				if err != nil {
					return nil, err
				}
				arguments := make([]*goExpression, len(expression.Args))
				for index, argument := range expression.Args {
					arguments[index], err = analyzeGoExpressionWithProgram(argument, signature, info, locals, functions, records, mutableLocals)
					if err != nil {
						return nil, err
					}
				}
				kind := goMethodCall
				if _, ok := selection.Recv().Underlying().(*types.Interface); ok {
					kind = goDynamicMethodCall
				}
				return &goExpression{kind: kind, left: receiver, methodID: methodID, arguments: arguments}, nil
			}
		}
		identifier, ok := ast.Unparen(expression.Fun).(*ast.Ident)
		if callSignature, signatureOK := goFunctionSignature(info.TypeOf(expression.Fun)); signatureOK && isUnaryI64Function(callSignature) {
			if !ok || functions[info.Uses[identifier]] == "" {
				callee, err := analyzeGoExpressionWithProgram(expression.Fun, signature, info, locals, functions, records, mutableLocals)
				if err != nil {
					return nil, err
				}
				if len(expression.Args) != 1 {
					return nil, fmt.Errorf("expression.indirect_call_arity")
				}
				argument, err := analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
				if err != nil {
					return nil, err
				}
				return &goExpression{kind: goIndirectCall, left: callee, arguments: []*goExpression{argument}}, nil
			}
		}
		if ok && len(expression.Args) == 1 {
			if typeName, typeOK := info.Uses[identifier].(*types.TypeName); typeOK {
				interfaceNamed, interfaceOK := typeName.Type().(*types.Named)
				if interfaceOK {
					_, contractOK := interfaceNamed.Underlying().(*types.Interface)
					concreteNamed, concreteOK := info.TypeOf(expression.Args[0]).(*types.Named)
					if contractOK && concreteOK {
						value, err := analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
						if err != nil {
							return nil, err
						}
						interfaceID := stableID("execution", "interface", interfaceNamed.Obj().Pkg().Path(), interfaceNamed.Obj().Name())
						concreteID := stableID("execution", "record", concreteNamed.Obj().Pkg().Path(), concreteNamed.Obj().Name())
						return &goExpression{kind: goInterfaceValue, left: value, typeID: interfaceID, witnessID: stableID("execution", "witness", concreteID, interfaceID)}, nil
					}
				}
			}
		}
		if ok && identifier.Name == "int" && info.Uses[identifier] == types.Universe.Lookup("int") && len(expression.Args) == 1 && isInt64(info.TypeOf(expression.Args[0])) {
			return analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
		}
		if ok && identifier.Name == "append" && info.Uses[identifier] == types.Universe.Lookup("append") && len(expression.Args) == 2 && !expression.Ellipsis.IsValid() && isI64Slice(info.TypeOf(expression.Args[0])) && isInt64(info.TypeOf(expression.Args[1])) {
			collection, err := analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			value, err := analyzeGoExpressionWithProgram(expression.Args[1], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: goCollectionAppend, left: collection, right: value}, nil
		}
		if ok && identifier.Name == "len" && info.Uses[identifier] == types.Universe.Lookup("len") && len(expression.Args) == 1 && !expression.Ellipsis.IsValid() {
			underlying := info.TypeOf(expression.Args[0]).Underlying()
			array, arrayOK := underlying.(*types.Array)
			slice, sliceOK := underlying.(*types.Slice)
			if (!arrayOK || !isInt64(array.Elem())) && (!sliceOK || !isInt64(slice.Elem())) {
				return nil, fmt.Errorf("expression.unsupported_collection_length")
			}
			collection, err := analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: goCollectionLength, left: collection}, nil
		}
		if selector, selectorOK := ast.Unparen(expression.Fun).(*ast.SelectorExpr); selectorOK && isSlicesFunction(info, selector, "Clone") && len(expression.Args) == 1 && !expression.Ellipsis.IsValid() && isI64Slice(info.TypeOf(expression.Args[0])) {
			return analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
		}
		if selector, selectorOK := ast.Unparen(expression.Fun).(*ast.SelectorExpr); selectorOK && isSlicesFunction(info, selector, "Replace") && len(expression.Args) == 4 && !expression.Ellipsis.IsValid() && isI64Slice(info.TypeOf(expression.Args[0])) && isInt64(info.TypeOf(expression.Args[3])) {
			indexName, indexOK := goI64IndexIdentifier(expression.Args[1], info)
			end, endOK := ast.Unparen(expression.Args[2]).(*ast.BinaryExpr)
			endIndex, endIndexOK := func() (*ast.Ident, bool) {
				if !endOK {
					return nil, false
				}
				return goI64IndexIdentifier(end.X, info)
			}()
			one, oneOK := func() (*ast.BasicLit, bool) {
				if !endOK {
					return nil, false
				}
				value, ok := ast.Unparen(end.Y).(*ast.BasicLit)
				return value, ok
			}()
			if !indexOK || !endIndexOK || !oneOK || end.Op != token.ADD || one.Kind != token.INT || one.Value != "1" || info.Uses[indexName] != info.Uses[endIndex] {
				return nil, fmt.Errorf("expression.collection_update_range")
			}
			collection, err := analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			index, err := analyzeGoExpressionWithProgram(expression.Args[1], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			value, err := analyzeGoExpressionWithProgram(expression.Args[3], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: goCollectionUpdate, left: collection, initial: index, right: value}, nil
		}
		callee, exists := functions[info.Uses[identifier]]
		if !ok || !exists || expression.Ellipsis.IsValid() {
			return nil, fmt.Errorf("expression.unsupported_call")
		}
		arguments := make([]*goExpression, len(expression.Args))
		calleeFunction, _ := info.Uses[identifier].(*types.Func)
		calleeSignature, _ := calleeFunction.Type().(*types.Signature)
		for index, argument := range expression.Args {
			analyzed, err := analyzeGoExpressionWithProgram(argument, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			if calleeSignature != nil && index < calleeSignature.Params().Len() {
				if interfaceNamed, ok := calleeSignature.Params().At(index).Type().(*types.Named); ok {
					if _, interfaceOK := interfaceNamed.Underlying().(*types.Interface); interfaceOK {
						if concreteNamed, concreteOK := info.TypeOf(argument).(*types.Named); concreteOK {
							interfaceID := stableID("execution", "interface", interfaceNamed.Obj().Pkg().Path(), interfaceNamed.Obj().Name())
							concreteID := stableID("execution", "record", concreteNamed.Obj().Pkg().Path(), concreteNamed.Obj().Name())
							analyzed = &goExpression{kind: goInterfaceValue, left: analyzed, typeID: interfaceID, witnessID: stableID("execution", "witness", concreteID, interfaceID)}
						}
					}
				}
			}
			arguments[index] = analyzed
		}
		return &goExpression{kind: goFunctionCall, callee: callee, arguments: arguments}, nil
	case *ast.FuncLit:
		closureSignature, ok := info.TypeOf(expression.Type).(*types.Signature)
		if !ok || !isUnaryI64Function(closureSignature) || len(expression.Body.List) != 1 {
			return nil, fmt.Errorf("expression.unsupported_closure")
		}
		statement, ok := expression.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(statement.Results) != 1 {
			return nil, fmt.Errorf("expression.unsupported_closure_body")
		}
		addition, ok := ast.Unparen(statement.Results[0]).(*ast.BinaryExpr)
		if !ok || addition.Op != token.ADD {
			return nil, fmt.Errorf("expression.unsupported_closure_body")
		}
		leftID, leftOK := ast.Unparen(addition.X).(*ast.Ident)
		rightID, rightOK := ast.Unparen(addition.Y).(*ast.Ident)
		if !leftOK || !rightOK {
			return nil, fmt.Errorf("expression.unsupported_closure_body")
		}
		captureIndex := -1
		for index := 0; index < signature.Params().Len(); index++ {
			if info.Uses[leftID] == signature.Params().At(index) {
				captureIndex = index
			}
		}
		if captureIndex < 0 || info.Uses[rightID] != closureSignature.Params().At(0) {
			return nil, fmt.Errorf("expression.unsupported_closure_capture")
		}
		return &goExpression{kind: goClosureConstruct, left: &goExpression{kind: goParameterRead, parameter: captureIndex}, body: &goExpression{kind: goIntegerAdd, left: &goExpression{kind: goCaptureRead}, right: &goExpression{kind: goClosureParameterRead}}, text: leftID.Name, elementName: rightID.Name, typeID: goFunctionTypeID(closureSignature)}, nil
	case *ast.CompositeLit:
		if isBytes(info.TypeOf(expression)) {
			bytes := make([]byte, len(expression.Elts))
			for index, element := range expression.Elts {
				value := info.Types[element].Value
				number, ok := constant.Uint64Val(value)
				if !ok || number > 255 {
					return nil, fmt.Errorf("expression.bytes_literal_element")
				}
				bytes[index] = byte(number)
			}
			return &goExpression{kind: goBytesLiteral, text: fmt.Sprintf("%x", bytes)}, nil
		}
		if _, ok := goOptionValueType(info.TypeOf(expression)); ok {
			fields, err := keyedCompositeFields(expression)
			if err != nil {
				return nil, err
			}
			typeID := goSemanticTypeIdentity(info.TypeOf(expression))
			if len(fields) == 0 {
				return &goExpression{kind: goOptionNone, typeID: typeID}, nil
			}
			present, ok := fields["Some"].(*ast.Ident)
			valueNode, hasValue := fields["Value"]
			if !ok || present.Name != "true" || !hasValue || len(fields) != 2 {
				return nil, fmt.Errorf("expression.option_constructor")
			}
			value, err := analyzeGoExpressionWithProgram(valueNode, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: goOptionSome, typeID: typeID, left: value}, nil
		}
		if _, _, ok := goResultTypes(info.TypeOf(expression)); ok {
			fields, err := keyedCompositeFields(expression)
			if err != nil {
				return nil, err
			}
			typeID := goSemanticTypeIdentity(info.TypeOf(expression))
			tag, ok := fields["Ok"].(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("expression.result_constructor")
			}
			field, kind := "Error", goResultError
			if tag.Name == "true" {
				field, kind = "Value", goResultOk
			} else if tag.Name != "false" {
				return nil, fmt.Errorf("expression.result_constructor")
			}
			valueNode, exists := fields[field]
			if !exists || len(fields) != 2 {
				return nil, fmt.Errorf("expression.result_constructor")
			}
			value, err := analyzeGoExpressionWithProgram(valueNode, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: kind, typeID: typeID, left: value}, nil
		}
		if stateType, resultType, ok := goTransitionTypes(info.TypeOf(expression)); ok {
			if len(expression.Elts) != 2 {
				return nil, fmt.Errorf("expression.transition_arity")
			}
			values := make([]ast.Expr, 2)
			for index, element := range expression.Elts {
				position := index
				value := element
				if keyed, keyedOK := element.(*ast.KeyValueExpr); keyedOK {
					name, nameOK := keyed.Key.(*ast.Ident)
					if !nameOK {
						return nil, fmt.Errorf("expression.transition_field")
					}
					if name.Name == "State" {
						position = 0
					} else if name.Name == "Result" {
						position = 1
					} else {
						return nil, fmt.Errorf("expression.transition_field")
					}
					value = keyed.Value
				}
				if values[position] != nil {
					return nil, fmt.Errorf("expression.transition_field")
				}
				values[position] = value
			}
			if values[0] == nil || values[1] == nil {
				return nil, fmt.Errorf("expression.transition_field")
			}
			state, err := analyzeGoExpressionWithProgram(values[0], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			result, err := analyzeGoExpressionWithProgram(values[1], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: goStateTransition, left: state, right: result, typeID: goTransitionTypeID(stateType, resultType)}, nil
		}
		if array, ok := info.TypeOf(expression).Underlying().(*types.Array); ok {
			if array.Len() < 0 || array.Len() > 32 || !isInt64(array.Elem()) || int64(len(expression.Elts)) != array.Len() {
				return nil, fmt.Errorf("expression.unsupported_fixed_array")
			}
			values := make([]*goExpression, len(expression.Elts))
			for index, element := range expression.Elts {
				if _, keyed := element.(*ast.KeyValueExpr); keyed {
					return nil, fmt.Errorf("expression.keyed_array_unsupported")
				}
				value, err := analyzeGoExpressionWithProgram(element, signature, info, locals, functions, records, mutableLocals)
				if err != nil {
					return nil, err
				}
				values[index] = value
			}
			length := uint64(array.Len())
			return &goExpression{kind: goFixedArrayConstruct, arrayType: stableID("execution", "type", "fixed-array", "i64", strconv.FormatUint(length, 10)), arrayLen: length, values: values}, nil
		}
		named, ok := info.TypeOf(expression).(*types.Named)
		record, exists := records[named]
		if !ok || !exists || len(expression.Elts) != len(record.ordered) {
			return nil, fmt.Errorf("expression.unsupported_record_construct")
		}
		values := make([]*goExpression, len(record.ordered))
		seen := make(map[*types.Var]bool, len(values))
		for sourceIndex, element := range expression.Elts {
			fieldIndex := sourceIndex
			valueExpression := element
			if keyed, keyedOK := element.(*ast.KeyValueExpr); keyedOK {
				identifier, identifierOK := keyed.Key.(*ast.Ident)
				fieldIndex = -1
				for index, field := range record.ordered {
					if identifierOK && field.Name() == identifier.Name {
						fieldIndex = index
						break
					}
				}
				if fieldIndex < 0 {
					return nil, fmt.Errorf("expression.unknown_record_field")
				}
				valueExpression = keyed.Value
			}
			field := record.ordered[fieldIndex]
			if seen[field] {
				return nil, fmt.Errorf("expression.duplicate_record_field")
			}
			seen[field] = true
			value, err := analyzeGoExpressionWithProgram(valueExpression, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			values[fieldIndex] = value
		}
		for _, value := range values {
			if value == nil {
				return nil, fmt.Errorf("expression.missing_record_field")
			}
		}
		return &goExpression{kind: goRecordConstruct, recordType: record.id, values: values}, nil
	case *ast.SelectorExpr:
		selection := info.Selections[expression]
		if selection == nil {
			return nil, fmt.Errorf("expression.unsupported_selector")
		}
		field, ok := selection.Obj().(*types.Var)
		if !ok {
			return nil, fmt.Errorf("expression.unsupported_selector")
		}
		if _, _, transition := goTransitionTypes(info.TypeOf(expression.X)); transition {
			value, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			if field.Name() == "State" {
				return &goExpression{kind: goTransitionState, left: value}, nil
			}
			if field.Name() == "Result" {
				return &goExpression{kind: goTransitionResult, left: value}, nil
			}
			return nil, fmt.Errorf("expression.unknown_transition_field")
		}
		var fieldID string
		for _, record := range records {
			if id, exists := record.fields[field]; exists {
				fieldID = id
				break
			}
		}
		if fieldID == "" {
			return nil, fmt.Errorf("expression.unknown_record_field")
		}
		record, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		return &goExpression{kind: goFieldRead, left: record, field: fieldID}, nil
	case *ast.IndexExpr:
		underlying := info.TypeOf(expression.X).Underlying()
		array, arrayOK := underlying.(*types.Array)
		slice, sliceOK := underlying.(*types.Slice)
		indexBasic, indexOK := info.TypeOf(expression.Index).Underlying().(*types.Basic)
		if ((!arrayOK || !isInt64(array.Elem())) && (!sliceOK || !isInt64(slice.Elem()))) || !indexOK || (indexBasic.Kind() != types.Int && indexBasic.Kind() != types.Int64) {
			return nil, fmt.Errorf("expression.unsupported_index_read")
		}
		collection, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		index, err := analyzeGoExpressionWithProgram(expression.Index, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		kind := goIndexRead
		if sliceOK {
			kind = goDynamicIndexRead
		}
		return &goExpression{kind: kind, left: collection, right: index}, nil
	default:
		return nil, fmt.Errorf("expression.unsupported_node")
	}
}

func isSlicesFunction(info *types.Info, selector *ast.SelectorExpr, name string) bool {
	function, ok := info.Uses[selector.Sel].(*types.Func)
	return ok && function.Name() == name && function.Pkg() != nil && function.Pkg().Path() == "slices"
}

func goI64IndexIdentifier(expression ast.Expr, info *types.Info) (*ast.Ident, bool) {
	if identifier, ok := ast.Unparen(expression).(*ast.Ident); ok {
		return identifier, isInt64(info.TypeOf(identifier))
	}
	conversion, ok := ast.Unparen(expression).(*ast.CallExpr)
	if !ok || len(conversion.Args) != 1 {
		return nil, false
	}
	typeName, ok := ast.Unparen(conversion.Fun).(*ast.Ident)
	identifier, identifierOK := ast.Unparen(conversion.Args[0]).(*ast.Ident)
	return identifier, ok && identifierOK && typeName.Name == "int" && info.Uses[typeName] == types.Universe.Lookup("int") && isInt64(info.TypeOf(identifier))
}

func isGoStringExpression(expression ast.Expr, info *types.Info) bool {
	if info == nil {
		return false
	}
	typeOf := info.TypeOf(expression)
	if typeOf == nil {
		return false
	}
	basic, ok := typeOf.Underlying().(*types.Basic)
	return ok && (basic.Kind() == types.String || basic.Kind() == types.UntypedString || basic.Info()&types.IsString != 0)
}

func evaluateBooleanExpression(expression *goExpression, parameters []int64) (bool, error) {
	if expression == nil {
		return false, fmt.Errorf("expression.nil")
	}
	switch expression.kind {
	case goBooleanLiteral:
		return expression.boolean, nil
	case goBooleanAnd:
		left, err := evaluateBooleanExpression(expression.left, parameters)
		if err != nil || !left {
			return false, err
		}
		return evaluateBooleanExpression(expression.right, parameters)
	case goBooleanOr:
		left, err := evaluateBooleanExpression(expression.left, parameters)
		if err != nil || left {
			return left, err
		}
		return evaluateBooleanExpression(expression.right, parameters)
	case goStringEqual:
		left, err := evaluateStringExpression(expression.left)
		if err != nil {
			return false, err
		}
		right, err := evaluateStringExpression(expression.right)
		if err != nil {
			return false, err
		}
		return left == right, nil
	case goIntegerLessEqual:
		left, err := evaluateIntegerExpression(expression.left, parameters)
		if err != nil {
			return false, err
		}
		right, err := evaluateIntegerExpression(expression.right, parameters)
		if err != nil {
			return false, err
		}
		return left <= right, nil
	default:
		return false, fmt.Errorf("expression.not_boolean")
	}
}

func evaluateStringExpression(expression *goExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("expression.nil")
	}
	switch expression.kind {
	case goStringLiteral:
		return expression.text, nil
	case goStringConcat:
		left, err := evaluateStringExpression(expression.left)
		if err != nil {
			return "", err
		}
		right, err := evaluateStringExpression(expression.right)
		if err != nil {
			return "", err
		}
		return left + right, nil
	default:
		return "", fmt.Errorf("expression.not_string")
	}
}

// evaluateIntegerExpression is the provider-neutral differential oracle for
// the bounded integer expression vocabulary. uint64 arithmetic makes the
// declared i64 modular overflow behavior explicit and independent of host
// signed-overflow rules.
func evaluateIntegerExpression(expression *goExpression, parameters []int64) (int64, error) {
	if expression == nil {
		return 0, fmt.Errorf("expression.nil")
	}
	switch expression.kind {
	case goParameterRead:
		if expression.parameter < 0 || expression.parameter >= len(parameters) {
			return 0, fmt.Errorf("expression.parameter_out_of_range")
		}
		return parameters[expression.parameter], nil
	case goIntegerLiteral:
		return int64(expression.integer), nil
	case goIntegerAdd, goIntegerMultiply, goIntegerSubtract:
		left, err := evaluateIntegerExpression(expression.left, parameters)
		if err != nil {
			return 0, err
		}
		right, err := evaluateIntegerExpression(expression.right, parameters)
		if err != nil {
			return 0, err
		}
		if expression.kind == goIntegerAdd {
			return int64(uint64(left) + uint64(right)), nil
		}
		if expression.kind == goIntegerMultiply {
			return int64(uint64(left) * uint64(right)), nil
		}
		return int64(uint64(left) - uint64(right)), nil
	default:
		return 0, fmt.Errorf("expression.not_integer")
	}
}

func matchAddParameters(expression *goExpression) (int, int, bool) {
	if expression == nil || expression.kind != goIntegerAdd || expression.left.kind != goParameterRead || expression.right.kind != goParameterRead {
		return 0, 0, false
	}
	return expression.left.parameter, expression.right.parameter, true
}

func matchAddLessEqualParameters(expression *goExpression) (decisionExpressionProfile, bool) {
	var profile decisionExpressionProfile
	if expression == nil || expression.kind != goIntegerLessEqual || expression.right.kind != goParameterRead {
		return profile, false
	}
	left, right, ok := matchAddParameters(expression.left)
	if !ok {
		return profile, false
	}
	limit := expression.right.parameter
	if left == right || left == limit || right == limit {
		return profile, false
	}
	return decisionExpressionProfile{addLeft: left, addRight: right, limit: limit, expression: expression}, true
}

func matchDecisionExpression(expression *goExpression, parameterCount int) bool {
	if expression == nil || expression.kind != goIntegerLessEqual {
		return false
	}
	seen := make(map[int]bool, parameterCount)
	var integer func(*goExpression) bool
	integer = func(expression *goExpression) bool {
		if expression == nil {
			return false
		}
		switch expression.kind {
		case goParameterRead:
			if expression.parameter < 0 || expression.parameter >= parameterCount {
				return false
			}
			seen[expression.parameter] = true
			return true
		case goIntegerLiteral:
			return true
		case goIntegerAdd, goIntegerMultiply, goIntegerSubtract:
			return integer(expression.left) && integer(expression.right)
		default:
			return false
		}
	}
	return integer(expression.left) && integer(expression.right) && len(seen) == parameterCount
}
