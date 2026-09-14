package goprovider

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

func goUnderlying(info *types.Info, expression ast.Expr) types.Type {
	if info == nil || expression == nil {
		return nil
	}
	typeOf := info.TypeOf(expression)
	if typeOf == nil {
		return nil
	}
	return typeOf.Underlying()
}

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
	goSliceConstruct
	goSliceRemove
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
	goBooleanNot
	goMutableClosureConstruct
	goStatefulIndirectCall
	goEmptyMap
	goMapLookup
	goMapLookupOption
	goMapUpdate
	goMapRemove
	goBytesLiteral
	goOptionNone
	goOptionSome
	goResultOk
	goResultError
	goVariantRead
	goOptionMatch
	goResultMatch
	goBytesEqual
	goUnitValue
	goNativeInvocation
	goNativeMethodInvocation
	goProductProject
	goNativeFieldRead
	goNativeBindingRead
	goNativeDefaultValue
	goNativeAddress
)

// goExpression is the provider's small typed source-expression tree. It keeps
// Go AST details out of canonical emission and normalizes equivalent source
// spellings before a target profile decides which tree shapes it supports.
type goExpression struct {
	kind              goExpressionKind
	parameter         int
	local             int
	integer           uint64
	boolean           bool
	text              string
	left              *goExpression
	right             *goExpression
	callee            string
	arguments         []*goExpression
	recordType        string
	field             string
	values            []*goExpression
	mutable           bool
	arrayType         string
	arrayLen          uint64
	initial           *goExpression
	body              *goExpression
	bindingID         string
	accName           string
	elementName       string
	receiverID        string
	methodID          string
	typeID            string
	witnessID         string
	alternate         *goExpression
	errorName         string
	errorTypeID       string
	nativeLanguage    string
	nativeTarget      string
	nativeSignature   string
	nativeResultType  string
	nativeResultTypes []string
	nativeTypes       map[string]string
	elementTypeID     string
	productIndex      uint64
	productTypes      []string
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
		case goUnitValue:
			typeID := stableID("execution", "type", "unit")
			id := expressionNodeID(owner, path, "unit")
			emitted[typeID] = graphEntity{typeID, entity(typeID, "0000000000000000000000000000a06a", nil)}
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a06b", []graphField{refField(0xa06b0, typeID)})}
			return id, nil
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
		case goNativeInvocation:
			if expression.nativeLanguage == "" || expression.nativeTarget == "" || expression.nativeSignature == "" || expression.nativeResultType == "" {
				return "", fmt.Errorf("expression.native_invocation_incomplete")
			}
			arguments := make([]string, len(expression.arguments))
			for index, argument := range expression.arguments {
				id, err := emit(argument, path+".argument."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
				arguments[index] = id
			}
			id := expressionNodeID(owner, path, "native-invocation")
			unitType := stableID("execution", "type", "unit")
			if expression.nativeResultType == unitType {
				emitted[unitType] = graphEntity{unitType, entity(unitType, "0000000000000000000000000000a06a", nil)}
			}
			if len(expression.nativeResultTypes) > 1 {
				emitted[expression.nativeResultType] = graphEntity{expression.nativeResultType, entity(expression.nativeResultType, "0000000000000000000000000000a06f", []graphField{refsField(0xa06f0, expression.nativeResultTypes)})}
			}
			nativeError := stableID("execution", "type", "native", "go", "error")
			for _, resultType := range expression.nativeResultTypes {
				if resultType == nativeError {
					emitted[nativeError] = graphEntity{nativeError, entity(nativeError, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, "go"), bytesField(0xa0711, "error")})}
				}
			}
			for nativeID, spelling := range expression.nativeTypes {
				emitted[nativeID] = graphEntity{nativeID, entity(nativeID, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, "go"), bytesField(0xa0711, spelling)})}
			}
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a06d", []graphField{
				bytesField(0xa06d0, expression.nativeLanguage), bytesField(0xa06d1, expression.nativeTarget),
				bytesField(0xa06d2, expression.nativeSignature), refsField(0xa06d3, arguments), refField(0xa06d4, expression.nativeResultType),
			})}
			return id, nil
		case goNativeDefaultValue:
			if expression.nativeLanguage == "" || expression.nativeResultType == "" {
				return "", fmt.Errorf("expression.native_default_incomplete")
			}
			for nativeID, spelling := range expression.nativeTypes {
				emitted[nativeID] = graphEntity{nativeID, entity(nativeID, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, expression.nativeLanguage), bytesField(0xa0711, spelling)})}
			}
			id := expressionNodeID(owner, path, "native-default")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a074", []graphField{
				bytesField(0xa0740, expression.nativeLanguage), refField(0xa0741, expression.nativeResultType),
			})}
			return id, nil
		case goNativeAddress:
			if expression.nativeLanguage == "" || expression.nativeResultType == "" || expression.left == nil {
				return "", fmt.Errorf("expression.native_address_incomplete")
			}
			operand, err := emit(expression.left, path+".operand")
			if err != nil {
				return "", err
			}
			for nativeID, spelling := range expression.nativeTypes {
				emitted[nativeID] = graphEntity{nativeID, entity(nativeID, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, expression.nativeLanguage), bytesField(0xa0711, spelling)})}
			}
			id := expressionNodeID(owner, path, "native-address")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a07a", []graphField{
				bytesField(0xa07a0, expression.nativeLanguage), refField(0xa07a1, operand), refField(0xa07a2, expression.nativeResultType),
			})}
			return id, nil
		case goNativeMethodInvocation:
			if expression.nativeLanguage == "" || expression.nativeTarget == "" || expression.nativeSignature == "" || expression.nativeResultType == "" || expression.left == nil {
				return "", fmt.Errorf("expression.native_method_invocation_incomplete")
			}
			receiver, err := emit(expression.left, path+".receiver")
			if err != nil {
				return "", err
			}
			arguments := make([]string, len(expression.arguments))
			for index, argument := range expression.arguments {
				id, err := emit(argument, path+".argument."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
				arguments[index] = id
			}
			id := expressionNodeID(owner, path, "native-method-invocation")
			unitType := stableID("execution", "type", "unit")
			if expression.nativeResultType == unitType {
				emitted[unitType] = graphEntity{unitType, entity(unitType, "0000000000000000000000000000a06a", nil)}
			}
			if len(expression.nativeResultTypes) > 1 {
				emitted[expression.nativeResultType] = graphEntity{expression.nativeResultType, entity(expression.nativeResultType, "0000000000000000000000000000a06f", []graphField{refsField(0xa06f0, expression.nativeResultTypes)})}
			}
			nativeError := stableID("execution", "type", "native", "go", "error")
			for _, resultType := range expression.nativeResultTypes {
				if resultType == nativeError {
					emitted[nativeError] = graphEntity{nativeError, entity(nativeError, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, "go"), bytesField(0xa0711, "error")})}
				}
			}
			for nativeID, spelling := range expression.nativeTypes {
				emitted[nativeID] = graphEntity{nativeID, entity(nativeID, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, "go"), bytesField(0xa0711, spelling)})}
			}
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a06e", []graphField{
				bytesField(0xa06e0, expression.nativeLanguage), bytesField(0xa06e1, expression.nativeTarget), bytesField(0xa06e2, expression.nativeSignature),
				refField(0xa06e3, receiver), refsField(0xa06e4, arguments), refField(0xa06e5, expression.nativeResultType),
			})}
			return id, nil
		case goProductProject:
			if expression.left == nil || expression.typeID == "" || expression.elementTypeID == "" || len(expression.productTypes) == 0 {
				return "", fmt.Errorf("expression.product_project_incomplete")
			}
			product, err := emit(expression.left, path+".product")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "product-project")
			emitted[expression.typeID] = graphEntity{expression.typeID, entity(expression.typeID, "0000000000000000000000000000a06f", []graphField{refsField(0xa06f0, expression.productTypes)})}
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a070", []graphField{
				refField(0xa0700, product), refField(0xa0701, expression.typeID), unsignedField(0xa0702, expression.productIndex), refField(0xa0703, expression.elementTypeID),
			})}
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
		case goNativeFieldRead:
			if expression.left == nil || expression.nativeLanguage == "" || expression.nativeTarget == "" || expression.nativeResultType == "" {
				return "", fmt.Errorf("expression.native_field_read_incomplete")
			}
			receiver, err := emit(expression.left, path+".receiver")
			if err != nil {
				return "", err
			}
			for nativeID, spelling := range expression.nativeTypes {
				emitted[nativeID] = graphEntity{nativeID, entity(nativeID, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, expression.nativeLanguage), bytesField(0xa0711, spelling)})}
			}
			id := expressionNodeID(owner, path, "native-field-read")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a072", []graphField{
				bytesField(0xa0720, expression.nativeLanguage), bytesField(0xa0721, expression.nativeTarget),
				refField(0xa0722, receiver), refField(0xa0723, expression.nativeResultType),
			})}
			return id, nil
		case goNativeBindingRead:
			if expression.nativeLanguage == "" || expression.nativeTarget == "" || expression.nativeResultType == "" {
				return "", fmt.Errorf("expression.native_binding_read_incomplete")
			}
			id := expressionNodeID(owner, path, "native-binding-read")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a073", []graphField{
				bytesField(0xa0730, expression.nativeLanguage), bytesField(0xa0731, expression.nativeTarget), refField(0xa0732, expression.nativeResultType),
			})}
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
		case goSliceConstruct:
			elementTypeID := expression.elementTypeID
			if elementTypeID == "" {
				elementTypeID = integerID
			}
			emitted[expression.typeID] = graphEntity{expression.typeID, entity(expression.typeID, "000000000000000000000000000090f8", []graphField{refField(0x9f80, elementTypeID)})}
			values := make([]string, len(expression.values))
			for index, value := range expression.values {
				var err error
				values[index], err = emit(value, path+".value."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
			}
			id := expressionNodeID(owner, path, "slice-construct")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a068", []graphField{refField(0xa0680, expression.typeID), refsField(0xa0681, values)})}
			return id, nil
		case goSliceRemove:
			collection, err := emit(expression.left, path+".collection")
			if err != nil {
				return "", err
			}
			index, err := emit(expression.right, path+".index")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "slice-remove")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a066", []graphField{refField(0xa0660, collection), refField(0xa0661, index)})}
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
		case goMapLookupOption:
			mapping, err := emit(expression.left, path+".map")
			if err != nil {
				return "", err
			}
			key, err := emit(expression.right, path+".key")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "map-lookup-option")
			emitted[expression.typeID] = graphEntity{expression.typeID, entity(expression.typeID, "0000000000000000000000000000a050", []graphField{refField(0xa0500, integerID)})}
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a044", []graphField{refField(0xa0440, mapping), refField(0xa0441, key), refField(0xa0442, expression.typeID)})}
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
		case goMapRemove:
			mapping, err := emit(expression.left, path+".map")
			if err != nil {
				return "", err
			}
			key, err := emit(expression.right, path+".key")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "map-remove")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a067", []graphField{refField(0xa0670, mapping), refField(0xa0671, key)})}
			return id, nil
		case goBytesLiteral:
			id := expressionNodeID(owner, path, "bytes-literal")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a064", []graphField{bytesHexField(0xa0640, expression.text)})}
			return id, nil
		case goBytesEqual:
			left, err := emit(expression.left, path+".left")
			if err != nil {
				return "", err
			}
			right, err := emit(expression.right, path+".right")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "bytes-equal")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a065", []graphField{refField(0xa0650, left), refField(0xa0651, right)})}
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
		case goBooleanNot:
			value, err := emit(expression.left, path+".value")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "boolean-not")
			emitted[id] = graphEntity{id, entity(id, "0000000000000000000000000000a069", []graphField{refField(0xa0690, value)})}
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
	if expression.kind == goOptionMatch || expression.kind == goResultMatch {
		return
	}
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
	id         string
	fields     map[*types.Var]string
	ordered    []*types.Var
	receiverID string
}

func analyzeGoExpressionWithProgram(expression ast.Expr, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutableLocals map[types.Object]bool) (*goExpression, error) {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		if signature.Recv() != nil && info.Uses[expression] == signature.Recv() {
			return &goExpression{kind: goReceiverRead, receiverID: goReceiverID(signature, records)}, nil
		}
		if local, ok := locals[info.Uses[expression]]; ok {
			kind := goLocalRead
			if mutableLocals[info.Uses[expression]] {
				kind = goPlaceRead
			}
			return &goExpression{kind: kind, local: local}, nil
		}
		for index := 0; index < signature.Params().Len(); index++ {
			if info.Uses[expression] == signature.Params().At(index) {
				return &goExpression{kind: goParameterRead, parameter: index}, nil
			}
		}
		if object, ok := info.Uses[expression].(*types.Const); ok {
			switch object.Val().Kind() {
			case constant.Int:
				value, exact := constant.Int64Val(object.Val())
				if !exact {
					return nil, fmt.Errorf("expression.constant_overflow")
				}
				return &goExpression{kind: goIntegerLiteral, integer: uint64(value)}, nil
			case constant.Bool:
				return &goExpression{kind: goBooleanLiteral, boolean: constant.BoolVal(object.Val())}, nil
			case constant.String:
				return &goExpression{kind: goStringLiteral, text: constant.StringVal(object.Val())}, nil
			}
		}
		if functions[nil] != "" {
			if object, ok := info.Uses[expression].(*types.Var); ok && object.Pkg() != nil && object.Parent() == object.Pkg().Scope() {
				resultTypeID, supported := goSupportedTypeID(object.Type(), stableID("execution", "type", "i64"), stableID("execution", "type", "bool"), stableID("execution", "type", "string"), records)
				nativeTypes := map[string]string(nil)
				if !supported && functions[nil] == "native-default" {
					if nativeID, native := goNativeTypeID(object.Type()); native {
						resultTypeID, supported = nativeID, true
						spelling, _ := goNativeTypeSpelling(object.Type())
						nativeTypes = map[string]string{nativeID: spelling}
					}
				}
				if supported {
					return &goExpression{kind: goNativeBindingRead, nativeLanguage: "go", nativeTarget: object.Pkg().Path() + "." + object.Name(), nativeResultType: resultTypeID, nativeTypes: nativeTypes}, nil
				}
			}
		}
		return nil, fmt.Errorf("expression.unresolved_parameter:%s", expression.Name)
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
		if (expression.Op == token.EQL || expression.Op == token.NEQ) && goErrorNilComparison(expression.X, expression.Y, info) {
			value := expression.X
			if isGoNil(expression.X) {
				value = expression.Y
			}
			analyzed, err := analyzeGoExpressionWithProgram(value, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			check := &goExpression{kind: goNativeInvocation, arguments: []*goExpression{analyzed}, nativeLanguage: "go", nativeTarget: "builtin.error.is_nil", nativeSignature: "func(error) bool", nativeResultType: stableID("execution", "type", "bool")}
			if expression.Op == token.NEQ {
				return &goExpression{kind: goBooleanNot, left: check}, nil
			}
			return check, nil
		}
		left, right := expression.X, expression.Y
		kind := goExpressionKind(0)
		negate := false
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
			if isGoStringExpression(left, info) && isGoStringExpression(right, info) {
				kind = goStringEqual
			} else if isGoIntegerExpression(left, info) && isGoIntegerExpression(right, info) && stableIntegerComparisonOperand(left, info) && stableIntegerComparisonOperand(right, info) {
				kind = goBooleanAnd
			} else {
				return analyzeNativeGoComparison(expression.Op, left, right, signature, info, locals, functions, records, mutableLocals)
			}
		case token.LEQ:
			kind = goIntegerLessEqual
		case token.GEQ:
			kind = goIntegerLessEqual
			left, right = right, left
		case token.LSS:
			kind = goIntegerLessEqual
			left, right, negate = right, left, true
		case token.GTR:
			if !isGoIntegerExpression(left, info) || !isGoIntegerExpression(right, info) || !stableIntegerComparisonOperand(left, info) || !stableIntegerComparisonOperand(right, info) {
				return analyzeNativeGoComparison(expression.Op, left, right, signature, info, locals, functions, records, mutableLocals)
			}
			kind = goIntegerLessEqual
			left, right, negate = left, right, true
		case token.NEQ:
			if isGoStringExpression(left, info) && isGoStringExpression(right, info) {
				kind = goStringEqual
				negate = true
			} else if !isGoIntegerExpression(left, info) || !isGoIntegerExpression(right, info) || !stableIntegerComparisonOperand(left, info) || !stableIntegerComparisonOperand(right, info) {
				return analyzeNativeGoComparison(expression.Op, left, right, signature, info, locals, functions, records, mutableLocals)
			}
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
		if expression.Op == token.NEQ {
			if kind == goStringEqual {
				return &goExpression{kind: goBooleanNot, left: &goExpression{kind: goStringEqual, left: analyzedLeft, right: analyzedRight}}, nil
			}
			equal := &goExpression{kind: goBooleanAnd,
				left:  &goExpression{kind: goIntegerLessEqual, left: analyzedLeft, right: analyzedRight},
				right: &goExpression{kind: goIntegerLessEqual, left: analyzedRight, right: analyzedLeft},
			}
			return &goExpression{kind: goBooleanNot, left: equal}, nil
		}
		var result *goExpression
		if expression.Op == token.EQL && kind == goBooleanAnd {
			result = &goExpression{kind: goBooleanAnd,
				left:  &goExpression{kind: goIntegerLessEqual, left: analyzedLeft, right: analyzedRight},
				right: &goExpression{kind: goIntegerLessEqual, left: analyzedRight, right: analyzedLeft},
			}
		} else {
			result = &goExpression{kind: kind, left: analyzedLeft, right: analyzedRight}
		}
		if negate {
			result = &goExpression{kind: goBooleanNot, left: result}
		}
		return result, nil
	case *ast.UnaryExpr:
		if expression.Op == token.AND && goExecutionModuleVersion(functions) >= 67 {
			value, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			resultType := info.TypeOf(expression)
			resultID, ok := goNativeTypeID(resultType)
			spelling, spellingOK := goNativeTypeSpelling(resultType)
			if !ok || !spellingOK {
				return nil, fmt.Errorf("expression.native_address_type")
			}
			return &goExpression{kind: goNativeAddress, left: value, nativeLanguage: "go", nativeResultType: resultID, nativeTypes: map[string]string{resultID: spelling}}, nil
		}
		if expression.Op != token.NOT {
			return nil, fmt.Errorf("expression.unsupported_unary_operator:%s", expression.Op)
		}
		value, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		return &goExpression{kind: goBooleanNot, left: value}, nil
	case *ast.TypeAssertExpr:
		if functions[nil] == "" || expression.Type == nil {
			return nil, fmt.Errorf("expression.unsupported_type_assertion")
		}
		tuple, commaOK := types.Unalias(info.TypeOf(expression)).(*types.Tuple)
		if !commaOK || tuple.Len() != 2 || !isBool(tuple.At(1).Type()) {
			return nil, fmt.Errorf("expression.unsupported_type_assertion")
		}
		assertedType := tuple.At(0).Type()
		assertedID, ok := goNativeTypeID(assertedType)
		if !ok {
			return nil, fmt.Errorf("expression.unsupported_type_assertion_target")
		}
		receiver, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		booleanID := stableID("execution", "type", "bool")
		productTypes := []string{assertedID, booleanID}
		productID := stableID("execution", "type", "product", assertedID, booleanID)
		spelling, _ := goNativeTypeSpelling(assertedType)
		return &goExpression{
			kind: goNativeInvocation, arguments: []*goExpression{receiver}, nativeLanguage: "go",
			nativeTarget: "builtin.type_assert_comma_ok[" + spelling + "]",
			nativeSignature: types.TypeString(tuple, func(pkg *types.Package) string {
				if pkg == nil {
					return ""
				}
				return pkg.Path()
			}),
			nativeResultType: productID, nativeResultTypes: productTypes,
			nativeTypes: map[string]string{assertedID: spelling},
		}, nil
	case *ast.CallExpr:
		if function, ok := ast.Unparen(expression.Fun).(*ast.FuncLit); ok {
			if match, valid := structuralTaggedMatchCall(function, expression, signature, info, locals, functions, records, mutableLocals); valid {
				return match, nil
			}
			if lookup, valid := structuralMapLookupOptionCall(function, expression, signature, info, locals, functions, records, mutableLocals); valid {
				return lookup, nil
			}
			kind, valid := structuralImmutableMapCall(function, expression, info)
			if !valid {
				return nil, fmt.Errorf("expression.unsupported_function_literal_call")
			}
			mapping, err := analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			key, err := analyzeGoExpressionWithProgram(expression.Args[1], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			if kind == goMapRemove {
				return &goExpression{kind: kind, left: mapping, right: key}, nil
			}
			value, err := analyzeGoExpressionWithProgram(expression.Args[2], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: kind, left: mapping, initial: key, right: value}, nil
		}
		if selector, ok := ast.Unparen(expression.Fun).(*ast.SelectorExpr); ok && isBytesFunction(info, selector, "Equal") && len(expression.Args) == 2 && !expression.Ellipsis.IsValid() {
			left, err := analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			right, err := analyzeGoExpressionWithProgram(expression.Args[1], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: goBytesEqual, left: left, right: right}, nil
		}
		if selector, ok := ast.Unparen(expression.Fun).(*ast.SelectorExpr); ok {
			if selection := info.Selections[selector]; selection != nil && selection.Kind() == types.MethodVal {
				methodID, exists := functions[selection.Obj()]
				if !exists {
					native, nativeErr := analyzeNativeGoMethodInvocation(selection, selector.X, expression.Args, expression.Ellipsis.IsValid(), signature, info, locals, functions, records, mutableLocals)
					if nativeErr == nil {
						return native, nil
					}
					return nil, fmt.Errorf("expression.unsupported_method_call:%v", nativeErr)
				}
				if expression.Ellipsis.IsValid() {
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
			// A selector without a Selection is a package-qualified function,
			// resolved by go/types to the same object collected from the local
			// source-module closure.
			if info.Selections[selector] == nil {
				if callee, exists := functions[info.Uses[selector.Sel]]; exists && !expression.Ellipsis.IsValid() {
					arguments := make([]*goExpression, len(expression.Args))
					for index, argument := range expression.Args {
						var err error
						arguments[index], err = analyzeGoExpressionWithProgram(argument, signature, info, locals, functions, records, mutableLocals)
						if err != nil {
							return nil, err
						}
					}
					return &goExpression{kind: goFunctionCall, callee: callee, arguments: arguments}, nil
				}
				if function, exists := info.Uses[selector.Sel].(*types.Func); exists {
					native, nativeErr := analyzeNativeGoInvocation(function, expression.Args, expression.Ellipsis.IsValid(), signature, info, locals, functions, records, mutableLocals)
					if nativeErr == nil {
						return native, nil
					}
					return nil, fmt.Errorf("expression.unsupported_call:%v", nativeErr)
				}
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
						if record, exists := findGoRecord(records, interfaceNamed); exists {
							interfaceID = record.id
						}
						concreteID := stableID("execution", "record", concreteNamed.Obj().Pkg().Path(), concreteNamed.Obj().Name())
						if record, exists := findGoRecord(records, concreteNamed); exists {
							concreteID = record.id
						}
						return &goExpression{kind: goInterfaceValue, left: value, typeID: interfaceID, witnessID: stableID("execution", "witness", concreteID, interfaceID)}, nil
					}
				}
			}
		}
		if ok && identifier.Name == "int" && info.Uses[identifier] == types.Universe.Lookup("int") && len(expression.Args) == 1 && isInt64(info.TypeOf(expression.Args[0])) {
			return analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
		}
		if ok && identifier.Name == "int64" && info.Uses[identifier] == types.Universe.Lookup("int64") && len(expression.Args) == 1 {
			if isInt64(info.TypeOf(expression.Args[0])) {
				return analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
			}
			if value := info.Types[expression.Args[0]].Value; value != nil && value.Kind() == constant.Int {
				if signed, exact := constant.Int64Val(value); exact {
					return &goExpression{kind: goIntegerLiteral, integer: uint64(signed)}, nil
				}
			}
			if call, yes := ast.Unparen(expression.Args[0]).(*ast.CallExpr); yes {
				if name, yes := ast.Unparen(call.Fun).(*ast.Ident); yes && name.Name == "len" && info.Uses[name] == types.Universe.Lookup("len") {
					return analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
				}
			}
		}
		if ok && identifier.Name == "append" && info.Uses[identifier] == types.Universe.Lookup("append") && len(expression.Args) == 2 && !expression.Ellipsis.IsValid() && isPrimitiveSlice(info.TypeOf(expression.Args[0])) && types.AssignableTo(info.TypeOf(expression.Args[1]), goUnderlying(info, expression.Args[0]).(*types.Slice).Elem()) {
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
			underlying := goUnderlying(info, expression.Args[0])
			array, arrayOK := underlying.(*types.Array)
			slice, sliceOK := underlying.(*types.Slice)
			if (!arrayOK || !isInt64(array.Elem())) && (!sliceOK || !(isInt64(slice.Elem()) || isBool(slice.Elem()) || isPureString(slice.Elem()))) {
				if functions[nil] != "native-default" {
					return nil, fmt.Errorf("expression.unsupported_collection_length")
				}
				collectionType := info.TypeOf(expression.Args[0])
				collection, err := analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
				if err != nil {
					return nil, err
				}
				collection, collectionID, materialized := nativeGoAssignmentValue(collection, collectionType)
				if !materialized {
					return nil, fmt.Errorf("expression.unsupported_collection_length")
				}
				collectionSpelling, _ := goNativeTypeSpelling(collectionType)
				intType := types.Universe.Lookup("int").Type()
				intID, _ := goNativeTypeID(intType)
				intSpelling, _ := goNativeTypeSpelling(intType)
				return &goExpression{
					kind: goNativeInvocation, arguments: []*goExpression{collection}, nativeLanguage: "go",
					nativeTarget:    "builtin.len[" + collectionSpelling + "]",
					nativeSignature: "func(" + collectionSpelling + ") int", nativeResultType: intID,
					nativeTypes: map[string]string{collectionID: collectionSpelling, intID: intSpelling},
				}, nil
			}
			collection, err := analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: goCollectionLength, left: collection}, nil
		}
		if selector, selectorOK := ast.Unparen(expression.Fun).(*ast.SelectorExpr); selectorOK && isSlicesFunction(info, selector, "Clone") && len(expression.Args) == 1 && !expression.Ellipsis.IsValid() && isPrimitiveSlice(info.TypeOf(expression.Args[0])) {
			return analyzeGoExpressionWithProgram(expression.Args[0], signature, info, locals, functions, records, mutableLocals)
		}
		if selector, selectorOK := ast.Unparen(expression.Fun).(*ast.SelectorExpr); selectorOK && isSlicesFunction(info, selector, "Replace") && len(expression.Args) == 4 && !expression.Ellipsis.IsValid() && isPrimitiveSlice(info.TypeOf(expression.Args[0])) && types.AssignableTo(info.TypeOf(expression.Args[3]), goUnderlying(info, expression.Args[0]).(*types.Slice).Elem()) {
			clone, cloneOK := ast.Unparen(expression.Args[0]).(*ast.CallExpr)
			cloneSelector, cloneSelectorOK := func() (*ast.SelectorExpr, bool) {
				if !cloneOK {
					return nil, false
				}
				value, ok := ast.Unparen(clone.Fun).(*ast.SelectorExpr)
				return value, ok
			}()
			if !cloneSelectorOK || !isSlicesFunction(info, cloneSelector, "Clone") || len(clone.Args) != 1 || clone.Ellipsis.IsValid() {
				return nil, fmt.Errorf("expression.collection_update_alias")
			}
			indexKey, indexOK := goI64IndexExpressionKey(expression.Args[1], info)
			end, endOK := ast.Unparen(expression.Args[2]).(*ast.BinaryExpr)
			endIndex, endIndexOK := func() (string, bool) {
				if !endOK {
					return "", false
				}
				return goI64IndexExpressionKey(end.X, info)
			}()
			one, oneOK := func() (*ast.BasicLit, bool) {
				if !endOK {
					return nil, false
				}
				value, ok := ast.Unparen(end.Y).(*ast.BasicLit)
				return value, ok
			}()
			if !indexOK || !endIndexOK || !oneOK || end.Op != token.ADD || one.Kind != token.INT || one.Value != "1" || indexKey != endIndex {
				return nil, fmt.Errorf("expression.collection_update_range")
			}
			collection, err := analyzeGoExpressionWithProgram(clone.Args[0], signature, info, locals, functions, records, mutableLocals)
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
		if selector, selectorOK := ast.Unparen(expression.Fun).(*ast.SelectorExpr); selectorOK && isSlicesFunction(info, selector, "Delete") && len(expression.Args) == 3 && !expression.Ellipsis.IsValid() && isPrimitiveSlice(info.TypeOf(expression.Args[0])) {
			clone, cloneOK := ast.Unparen(expression.Args[0]).(*ast.CallExpr)
			cloneSelector, cloneSelectorOK := func() (*ast.SelectorExpr, bool) {
				if !cloneOK {
					return nil, false
				}
				value, ok := ast.Unparen(clone.Fun).(*ast.SelectorExpr)
				return value, ok
			}()
			if !cloneSelectorOK || !isSlicesFunction(info, cloneSelector, "Clone") || len(clone.Args) != 1 || clone.Ellipsis.IsValid() {
				return nil, fmt.Errorf("expression.slice_remove_alias")
			}
			indexKey, indexOK := goI64IndexExpressionKey(expression.Args[1], info)
			end, endOK := ast.Unparen(expression.Args[2]).(*ast.BinaryExpr)
			endIndex, endIndexOK := func() (string, bool) {
				if !endOK {
					return "", false
				}
				return goI64IndexExpressionKey(end.X, info)
			}()
			one, oneOK := func() (*ast.BasicLit, bool) {
				if !endOK {
					return nil, false
				}
				value, ok := ast.Unparen(end.Y).(*ast.BasicLit)
				return value, ok
			}()
			if !indexOK || !endIndexOK || !oneOK || end.Op != token.ADD || one.Kind != token.INT || one.Value != "1" || indexKey != endIndex {
				return nil, fmt.Errorf("expression.slice_remove_range")
			}
			collection, err := analyzeGoExpressionWithProgram(clone.Args[0], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			index, err := analyzeGoExpressionWithProgram(expression.Args[1], signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			return &goExpression{kind: goSliceRemove, left: collection, right: index}, nil
		}
		if ok {
			if _, builtin := info.Uses[identifier].(*types.Builtin); builtin {
				if invocation, builtinErr := analyzeNativeGoBuiltinCall(identifier.Name, expression, signature, info, locals, functions, records, mutableLocals); builtinErr == nil {
					return invocation, nil
				}
			}
		}
		callee, exists := functions[info.Uses[identifier]]
		if !ok || !exists || expression.Ellipsis.IsValid() {
			if ok {
				if function, native := info.Uses[identifier].(*types.Func); native {
					invocation, nativeErr := analyzeNativeGoInvocation(function, expression.Args, expression.Ellipsis.IsValid(), signature, info, locals, functions, records, mutableLocals)
					if nativeErr == nil {
						return invocation, nil
					}
					return nil, fmt.Errorf("expression.unsupported_call:%v", nativeErr)
				}
			}
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
							if record, exists := findGoRecord(records, interfaceNamed); exists {
								interfaceID = record.id
							}
							concreteID := stableID("execution", "record", concreteNamed.Obj().Pkg().Path(), concreteNamed.Obj().Name())
							if record, exists := findGoRecord(records, concreteNamed); exists {
								concreteID = record.id
							}
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
		captureName, parameterName := leftID, rightID
		for index := 0; index < signature.Params().Len(); index++ {
			if info.Uses[leftID] == signature.Params().At(index) {
				captureIndex = index
			}
		}
		if captureIndex < 0 || info.Uses[rightID] != closureSignature.Params().At(0) {
			captureIndex = -1
			captureName, parameterName = rightID, leftID
			for index := 0; index < signature.Params().Len(); index++ {
				if info.Uses[rightID] == signature.Params().At(index) {
					captureIndex = index
				}
			}
		}
		if captureIndex < 0 || info.Uses[parameterName] != closureSignature.Params().At(0) {
			return nil, fmt.Errorf("expression.unsupported_closure_capture")
		}
		return &goExpression{kind: goClosureConstruct, left: &goExpression{kind: goParameterRead, parameter: captureIndex}, body: &goExpression{kind: goIntegerAdd, left: &goExpression{kind: goCaptureRead}, right: &goExpression{kind: goClosureParameterRead}}, text: captureName.Name, elementName: parameterName.Name, typeID: goFunctionTypeID(closureSignature)}, nil
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
			if !ok && fields["Ok"] == nil {
				tag, ok = &ast.Ident{Name: "false"}, true
			}
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
			expectedFields := 2
			if tag.Name == "false" && fields["Ok"] == nil {
				expectedFields = 1
			}
			if !exists || len(fields) != expectedFields {
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
		if array, ok := goUnderlying(info, expression).(*types.Array); ok {
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
		if slice, ok := goUnderlying(info, expression).(*types.Slice); ok && (isInt64(slice.Elem()) || isBool(slice.Elem()) || isPureString(slice.Elem())) {
			if len(expression.Elts) > 512 {
				return nil, fmt.Errorf("expression.slice_construct_bounds")
			}
			values := make([]*goExpression, len(expression.Elts))
			for index, element := range expression.Elts {
				if _, keyed := element.(*ast.KeyValueExpr); keyed {
					return nil, fmt.Errorf("expression.keyed_slice_unsupported")
				}
				value, err := analyzeGoExpressionWithProgram(element, signature, info, locals, functions, records, mutableLocals)
				if err != nil {
					return nil, err
				}
				values[index] = value
			}
			elementTypeID := goSemanticTypeIdentity(slice.Elem())
			tag, _ := goPrimitiveTypeTag(slice.Elem())
			return &goExpression{kind: goSliceConstruct, typeID: stableID("execution", "type", "slice", tag), elementTypeID: elementTypeID, values: values}, nil
		}
		if isI64Map(info.TypeOf(expression)) {
			if len(expression.Elts) != 0 {
				return nil, fmt.Errorf("expression.map_literal_nonempty")
			}
			return &goExpression{kind: goEmptyMap, typeID: stableID("execution", "type", "map", "i64", "i64")}, nil
		}
		named, ok := types.Unalias(info.TypeOf(expression)).(*types.Named)
		record, exists := findGoRecord(records, named)
		if !ok || !exists {
			if functions[nil] == "native-default" {
				nativeType := info.TypeOf(expression)
				nativeID, native := goNativeTypeID(nativeType)
				spelling, spellingOK := goNativeTypeSpelling(nativeType)
				if native && spellingOK {
					arguments := make([]*goExpression, 0, len(expression.Elts)*2)
					shape := make([]string, 0, len(expression.Elts))
					_, structLiteral := goUnderlying(info, expression).(*types.Struct)
					for _, element := range expression.Elts {
						valueNode := element
						if keyed, keyedOK := element.(*ast.KeyValueExpr); keyedOK {
							valueNode = keyed.Value
							if identifier, identifierOK := keyed.Key.(*ast.Ident); structLiteral && identifierOK {
								shape = append(shape, "field:"+identifier.Name)
							} else {
								key, err := analyzeGoExpressionWithProgram(keyed.Key, signature, info, locals, functions, records, mutableLocals)
								if err != nil {
									return nil, err
								}
								shape = append(shape, "key")
								arguments = append(arguments, key)
							}
						} else {
							shape = append(shape, "position")
						}
						value, err := analyzeGoExpressionWithProgram(valueNode, signature, info, locals, functions, records, mutableLocals)
						if err != nil {
							return nil, err
						}
						arguments = append(arguments, value)
					}
					return &goExpression{
						kind: goNativeInvocation, arguments: arguments, nativeLanguage: "go",
						nativeTarget:    "builtin.composite_literal[" + spelling + ";" + strings.Join(shape, ",") + "]",
						nativeSignature: "func(...) " + spelling, nativeResultType: nativeID,
						nativeTypes: map[string]string{nativeID: spelling},
					}, nil
				}
			}
			return nil, fmt.Errorf("expression.unsupported_record_construct")
		}
		values := make([]*goExpression, len(record.ordered))
		seen := make(map[*types.Var]bool, len(values))
		keyedLiteral := false
		for sourceIndex, element := range expression.Elts {
			fieldIndex := sourceIndex
			valueExpression := element
			if keyed, keyedOK := element.(*ast.KeyValueExpr); keyedOK {
				keyedLiteral = true
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
			} else if fieldIndex >= len(record.ordered) {
				return nil, fmt.Errorf("expression.record_field_count")
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
		for index, value := range values {
			if value == nil && keyedLiteral {
				var err error
				values[index], err = zeroGoExpression(record.ordered[index].Type(), records, map[string]bool{}, 0)
				if err != nil && functions[nil] == "native-default" && named.Obj() != nil && named.Obj().Pkg() != nil && goTypeOwnedOutsidePackage(record.ordered[index].Type(), named.Obj().Pkg().Path()) {
					if nativeID, ok := goNativeTypeID(record.ordered[index].Type()); ok {
						spelling, _ := goNativeTypeSpelling(record.ordered[index].Type())
						values[index] = &goExpression{kind: goNativeDefaultValue, nativeLanguage: "go", nativeResultType: nativeID, nativeTypes: map[string]string{nativeID: spelling}}
						err = nil
					}
				}
				if err != nil {
					return nil, err
				}
			} else if value == nil {
				return nil, fmt.Errorf("expression.missing_record_field")
			}
		}
		return &goExpression{kind: goRecordConstruct, recordType: record.id, values: values}, nil
	case *ast.SelectorExpr:
		selection := info.Selections[expression]
		if selection == nil {
			qualifier, qualified := ast.Unparen(expression.X).(*ast.Ident)
			_, imported := info.Uses[qualifier].(*types.PkgName)
			object, constantOK := info.Uses[expression.Sel].(*types.Const)
			if qualified && imported && constantOK {
				switch object.Val().Kind() {
				case constant.Int:
					value, exact := constant.Int64Val(object.Val())
					if !exact {
						return nil, fmt.Errorf("expression.imported_constant_overflow")
					}
					return &goExpression{kind: goIntegerLiteral, integer: uint64(value)}, nil
				case constant.Bool:
					return &goExpression{kind: goBooleanLiteral, boolean: constant.BoolVal(object.Val())}, nil
				case constant.String:
					return &goExpression{kind: goStringLiteral, text: constant.StringVal(object.Val())}, nil
				default:
					return nil, fmt.Errorf("expression.imported_constant_kind")
				}
			}
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
		if item, option := goOptionValueType(info.TypeOf(expression.X)); option {
			value, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			match := &goExpression{kind: goOptionMatch, left: value, text: "value", typeID: goSemanticTypeIdentity(item)}
			switch field.Name() {
			case "Some":
				match.initial = &goExpression{kind: goBooleanLiteral}
				match.body = &goExpression{kind: goBooleanLiteral, boolean: true}
			case "Value":
				zero, err := zeroGoExpression(item, records, map[string]bool{}, 0)
				if err != nil {
					return nil, err
				}
				match.initial = zero
				match.body = &goExpression{kind: goVariantRead}
			default:
				return nil, fmt.Errorf("expression.unknown_option_field")
			}
			return match, nil
		}
		if success, failure, result := goResultTypes(info.TypeOf(expression.X)); result {
			value, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			match := &goExpression{kind: goResultMatch, left: value, text: "value", typeID: goSemanticTypeIdentity(success), errorName: "failure", errorTypeID: goSemanticTypeIdentity(failure)}
			switch field.Name() {
			case "Ok":
				match.body = &goExpression{kind: goBooleanLiteral, boolean: true}
				match.alternate = &goExpression{kind: goBooleanLiteral}
			case "Value":
				match.body = &goExpression{kind: goVariantRead}
				zero, err := zeroGoExpression(success, records, map[string]bool{}, 0)
				if err != nil {
					return nil, err
				}
				match.alternate = zero
			case "Error":
				zero, err := zeroGoExpression(failure, records, map[string]bool{}, 0)
				if err != nil {
					return nil, err
				}
				match.body = zero
				match.alternate = &goExpression{kind: goVariantRead}
			default:
				return nil, fmt.Errorf("expression.unknown_result_field")
			}
			return match, nil
		}
		var fieldID string
		if receiver, ok := types.Unalias(info.TypeOf(expression.X)).(*types.Named); ok {
			if record, exists := findGoRecord(records, receiver); exists {
				for candidate, id := range record.fields {
					if candidate.Name() == field.Name() {
						fieldID = id
						break
					}
				}
			}
		}
		if fieldID == "" {
			ownerPath := ""
			for object := range functions {
				if object != nil && object.Pkg() != nil {
					ownerPath = object.Pkg().Path()
					break
				}
			}
			receiverType := info.TypeOf(expression.X)
			resultType := info.TypeOf(expression)
			resultTypeID, resultOK := goSupportedTypeID(resultType, stableID("execution", "type", "i64"), stableID("execution", "type", "bool"), stableID("execution", "type", "string"), records)
			nativeTypes := map[string]string(nil)
			if !resultOK && ownerPath != "" && goTypeOwnedOutsidePackage(resultType, ownerPath) {
				if nativeID, ok := goNativeTypeID(resultType); ok {
					resultTypeID, resultOK = nativeID, true
					spelling, _ := goNativeTypeSpelling(resultType)
					nativeTypes = map[string]string{nativeID: spelling}
				}
			}
			_, receiverNative := goNativeTypeID(receiverType)
			if functions[nil] != "" && ownerPath != "" && receiverNative && resultOK {
				receiver, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
				if err != nil {
					return nil, err
				}
				target := types.TypeString(receiverType, func(pkg *types.Package) string {
					if pkg == nil {
						return ""
					}
					return pkg.Path()
				}) + "." + field.Name()
				return &goExpression{kind: goNativeFieldRead, left: receiver, nativeLanguage: "go", nativeTarget: target, nativeResultType: resultTypeID, nativeTypes: nativeTypes}, nil
			}
			return nil, fmt.Errorf("expression.unknown_record_field:%s:%s", types.TypeString(receiverType, nil), field.Name())
		}
		record, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		return &goExpression{kind: goFieldRead, left: record, field: fieldID}, nil
	case *ast.IndexExpr:
		underlying := goUnderlying(info, expression.X)
		array, arrayOK := underlying.(*types.Array)
		slice, sliceOK := underlying.(*types.Slice)
		mapping, mapOK := underlying.(*types.Map)
		indexBasic, indexOK := goUnderlying(info, expression.Index).(*types.Basic)
		if ((!arrayOK || !isInt64(array.Elem())) && (!sliceOK || !(isInt64(slice.Elem()) || isBool(slice.Elem()) || isPureString(slice.Elem()))) && (!mapOK || !isInt64(mapping.Key()) || !isInt64(mapping.Elem()))) || !indexOK || (indexBasic.Kind() != types.Int && indexBasic.Kind() != types.Int64) {
			if goExecutionModuleVersion(functions) >= 59 && functions[nil] == "native-default" {
				return analyzeNativeGoIndex(expression, signature, info, locals, functions, records, mutableLocals)
			}
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
		} else if mapOK {
			kind = goMapLookup
		}
		return &goExpression{kind: kind, left: collection, right: index}, nil
	default:
		return nil, fmt.Errorf("expression.unsupported_node")
	}
}

func analyzeNativeGoBuiltinCall(name string, call *ast.CallExpr, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutableLocals map[types.Object]bool) (*goExpression, error) {
	if goExecutionModuleVersion(functions) < 60 || functions[nil] != "native-default" {
		return nil, fmt.Errorf("expression.native_builtin_disabled")
	}
	var expected []types.Type
	resultType := info.TypeOf(call)
	switch name {
	case "make":
		if len(call.Args) < 1 || resultType == nil {
			return nil, fmt.Errorf("expression.native_builtin_shape")
		}
		intType := types.Universe.Lookup("int").Type()
		for range call.Args[1:] {
			expected = append(expected, intType)
		}
	case "append":
		if len(call.Args) < 2 || resultType == nil {
			return nil, fmt.Errorf("expression.native_builtin_shape")
		}
		slice, ok := resultType.Underlying().(*types.Slice)
		if !ok {
			return nil, fmt.Errorf("expression.native_builtin_shape")
		}
		expected = append(expected, resultType)
		for index := 1; index < len(call.Args); index++ {
			valueType := slice.Elem()
			if call.Ellipsis.IsValid() && index == len(call.Args)-1 {
				valueType = resultType
			}
			expected = append(expected, valueType)
		}
	case "delete":
		if len(call.Args) != 2 {
			return nil, fmt.Errorf("expression.native_builtin_shape")
		}
		mapping, ok := info.TypeOf(call.Args[0]).Underlying().(*types.Map)
		if !ok {
			return nil, fmt.Errorf("expression.native_builtin_shape")
		}
		expected = []types.Type{info.TypeOf(call.Args[0]), mapping.Key()}
		resultType = nil
	case "copy":
		if len(call.Args) != 2 {
			return nil, fmt.Errorf("expression.native_builtin_shape")
		}
		expected = []types.Type{info.TypeOf(call.Args[0]), info.TypeOf(call.Args[1])}
	case "new":
		if len(call.Args) != 1 || resultType == nil {
			return nil, fmt.Errorf("expression.native_builtin_shape")
		}
		expected = nil
	case "cap":
		if len(call.Args) != 1 || resultType == nil {
			return nil, fmt.Errorf("expression.native_builtin_shape")
		}
		expected = []types.Type{info.TypeOf(call.Args[0])}
	case "clear":
		if len(call.Args) != 1 {
			return nil, fmt.Errorf("expression.native_builtin_shape")
		}
		expected = []types.Type{info.TypeOf(call.Args[0])}
		resultType = nil
	default:
		return nil, fmt.Errorf("expression.native_builtin_unsupported")
	}
	start := 0
	if name == "make" || name == "new" {
		start = 1
	}
	arguments := make([]*goExpression, 0, len(expected))
	nativeTypes := map[string]string{}
	for index, target := range expected {
		node := call.Args[start+index]
		value, typeID, spelling, err := analyzeNativeGoOperand(node, target, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, value)
		nativeTypes[typeID] = spelling
	}
	resultID := stableID("execution", "type", "unit")
	resultSpelling := "unit"
	if resultType != nil {
		resultID, _ = goSupportedTypeID(resultType, stableID("execution", "type", "i64"), stableID("execution", "type", "bool"), stableID("execution", "type", "string"), records)
		resultSpelling = types.TypeString(resultType, nil)
		if resultID == "" {
			var ok bool
			resultID, ok = goNativeTypeID(resultType)
			if !ok {
				return nil, fmt.Errorf("expression.native_builtin_result")
			}
			resultSpelling, _ = goNativeTypeSpelling(resultType)
			nativeTypes[resultID] = resultSpelling
		}
	}
	target := "builtin." + name + "[" + resultSpelling + "]"
	if call.Ellipsis.IsValid() {
		target += ".ellipsis"
	}
	return &goExpression{kind: goNativeInvocation, arguments: arguments, nativeLanguage: "go", nativeTarget: target, nativeSignature: types.TypeString(info.TypeOf(call.Fun), nil), nativeResultType: resultID, nativeTypes: nativeTypes}, nil
}

func analyzeNativeGoIndex(expression *ast.IndexExpr, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutableLocals map[types.Object]bool) (*goExpression, error) {
	collectionType := info.TypeOf(expression.X)
	resultType := info.TypeOf(expression)
	if collectionType == nil || resultType == nil {
		return nil, fmt.Errorf("expression.unsupported_index_read")
	}
	var indexType types.Type
	switch collectionType.Underlying().(type) {
	case *types.Array, *types.Slice:
		indexType = types.Universe.Lookup("int").Type()
	case *types.Map:
		indexType = collectionType.Underlying().(*types.Map).Key()
	case *types.Basic:
		basic := collectionType.Underlying().(*types.Basic)
		if basic.Kind() != types.String {
			return nil, fmt.Errorf("expression.unsupported_index_read")
		}
		indexType = types.Universe.Lookup("int").Type()
	default:
		return nil, fmt.Errorf("expression.unsupported_index_read")
	}
	collection, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
	if err != nil {
		return nil, err
	}
	collection, collectionID, collectionOK := nativeGoAssignmentValue(collection, collectionType)
	index, indexID, indexSpelling, err := analyzeNativeGoOperand(expression.Index, indexType, signature, info, locals, functions, records, mutableLocals)
	if err != nil || !collectionOK {
		return nil, fmt.Errorf("expression.unsupported_index_read")
	}
	resultID, resultOK := goSupportedTypeID(resultType, stableID("execution", "type", "i64"), stableID("execution", "type", "bool"), stableID("execution", "type", "string"), records)
	nativeTypes := map[string]string{indexID: indexSpelling}
	collectionSpelling, _ := goNativeTypeSpelling(collectionType)
	nativeTypes[collectionID] = collectionSpelling
	resultSpelling := types.TypeString(resultType, nil)
	if !resultOK {
		resultID, resultOK = goNativeTypeID(resultType)
		if spelling, native := goNativeTypeSpelling(resultType); native {
			resultSpelling = spelling
			nativeTypes[resultID] = spelling
		}
	}
	if !resultOK {
		return nil, fmt.Errorf("expression.unsupported_index_read")
	}
	return &goExpression{kind: goNativeInvocation, arguments: []*goExpression{collection, index}, nativeLanguage: "go", nativeTarget: "builtin.index[" + collectionSpelling + "]", nativeSignature: "func(" + collectionSpelling + ", " + indexSpelling + ") " + resultSpelling, nativeResultType: resultID, nativeTypes: nativeTypes}, nil
}

func analyzeNativeGoComparison(operator token.Token, leftNode, rightNode ast.Expr, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutableLocals map[types.Object]bool) (*goExpression, error) {
	if functions[nil] != "native-default" {
		if operator == token.NEQ {
			return nil, fmt.Errorf("expression.unsupported_integer_not_equal_operands")
		}
		return nil, fmt.Errorf("expression.unsupported_operator:%s", operator)
	}
	leftType, rightType := info.TypeOf(leftNode), info.TypeOf(rightNode)
	leftExpected, rightExpected := leftType, rightType
	if goUntypedType(leftType) {
		leftExpected = rightType
	}
	if goUntypedType(rightType) {
		rightExpected = leftType
	}
	left, leftID, leftSpelling, err := analyzeNativeGoOperand(leftNode, leftExpected, signature, info, locals, functions, records, mutableLocals)
	if err != nil {
		return nil, err
	}
	right, rightID, rightSpelling, err := analyzeNativeGoOperand(rightNode, rightExpected, signature, info, locals, functions, records, mutableLocals)
	if err != nil {
		return nil, err
	}
	typesByID := map[string]string{}
	if leftID != "" {
		typesByID[leftID] = leftSpelling
	}
	if rightID != "" {
		typesByID[rightID] = rightSpelling
	}
	return &goExpression{
		kind: goNativeInvocation, arguments: []*goExpression{left, right}, nativeLanguage: "go",
		nativeTarget:     "builtin.compare[" + operator.String() + ";" + leftSpelling + ";" + rightSpelling + "]",
		nativeSignature:  "func(" + leftSpelling + ", " + rightSpelling + ") bool",
		nativeResultType: stableID("execution", "type", "bool"), nativeTypes: typesByID,
	}, nil
}

func analyzeNativeGoOperand(node ast.Expr, expected types.Type, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutableLocals map[types.Object]bool) (*goExpression, string, string, error) {
	spelling, native := goNativeTypeSpelling(expected)
	if !native {
		return nil, "", "", fmt.Errorf("expression.native_comparison_type")
	}
	typeID, _ := goNativeTypeID(expected)
	if identifier, nilValue := ast.Unparen(node).(*ast.Ident); nilValue && identifier.Name == "nil" {
		return &goExpression{kind: goNativeDefaultValue, nativeLanguage: "go", nativeResultType: typeID, nativeTypes: map[string]string{typeID: spelling}}, typeID, spelling, nil
	}
	value, err := analyzeGoExpressionWithProgram(node, signature, info, locals, functions, records, mutableLocals)
	if err != nil {
		return nil, "", "", err
	}
	value, _, ok := nativeGoAssignmentValue(value, expected)
	if !ok {
		return nil, "", "", fmt.Errorf("expression.native_comparison_type")
	}
	return value, typeID, spelling, nil
}

func goUntypedType(value types.Type) bool {
	basic, ok := types.Unalias(value).(*types.Basic)
	return ok && basic.Info()&types.IsUntyped != 0
}

// analyzeNativeGoInvocation retains a typed call to a Go-runtime callable as an
// explicit realization boundary. The surrounding function remains canonical;
// targets that cannot provide the declared Go mechanic must reject placement.
func analyzeNativeGoInvocation(function *types.Func, argumentsAST []ast.Expr, ellipsis bool, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutableLocals map[types.Object]bool) (*goExpression, error) {
	if function == nil || function.Pkg() == nil || ellipsis {
		return nil, fmt.Errorf("expression.native_call_target")
	}
	callSignature, ok := function.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("expression.native_call_signature")
	}
	resultType, resultTypes, nativeTypes, ok := nativeGoResultTypeID(callSignature, records)
	if !ok {
		return nil, fmt.Errorf("expression.native_call_result_type")
	}
	arguments := make([]*goExpression, len(argumentsAST))
	for index, argument := range argumentsAST {
		analyzed, err := analyzeGoExpressionWithProgram(argument, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		arguments[index] = analyzed
	}
	target := function.Pkg().Path() + "." + function.Name()
	return &goExpression{kind: goNativeInvocation, arguments: arguments, nativeLanguage: "go", nativeTarget: target, nativeSignature: types.TypeString(callSignature, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	}), nativeResultType: resultType, nativeResultTypes: resultTypes, nativeTypes: nativeTypes}, nil
}

func nativeGoResultTypeID(signature *types.Signature, records map[*types.Named]goRecordInfo) (string, []string, map[string]string, bool) {
	if signature == nil {
		return "", nil, nil, false
	}
	if signature.Results().Len() == 0 {
		return stableID("execution", "type", "unit"), nil, nil, true
	}
	if signature.Results().Len() > 1 {
		items := make([]string, signature.Results().Len())
		native := map[string]string{}
		parts := []string{"execution", "type", "product"}
		for index := range items {
			value := signature.Results().At(index).Type()
			item, ok := goSupportedTypeID(value, stableID("execution", "type", "i64"), stableID("execution", "type", "bool"), stableID("execution", "type", "string"), records)
			if !ok {
				spelling, nativeOK := goNativeTypeSpelling(value)
				if !nativeOK {
					return "", nil, nil, false
				}
				item = stableID("execution", "type", "native", "go", spelling)
				native[item] = spelling
			}
			items[index] = item
			parts = append(parts, item)
		}
		return stableID(parts...), items, native, true
	}
	result := signature.Results().At(0).Type()
	switch {
	case isInt64(result):
		return stableID("execution", "type", "i64"), nil, nil, true
	case isBool(result):
		return stableID("execution", "type", "bool"), nil, nil, true
	case isPureString(result):
		return stableID("execution", "type", "string"), nil, nil, true
	case isGoErrorType(result):
		id := stableID("execution", "type", "native", "go", "error")
		return id, []string{id}, map[string]string{id: "error"}, true
	default:
		spelling, ok := goNativeTypeSpelling(result)
		if !ok {
			return "", nil, nil, false
		}
		id := stableID("execution", "type", "native", "go", spelling)
		return id, []string{id}, map[string]string{id: spelling}, true
	}
}

func isGoNil(expression ast.Expr) bool {
	identifier, ok := ast.Unparen(expression).(*ast.Ident)
	return ok && identifier.Name == "nil"
}

func goErrorNilComparison(left, right ast.Expr, info *types.Info) bool {
	return isGoNil(left) && isGoErrorType(info.TypeOf(right)) || isGoNil(right) && isGoErrorType(info.TypeOf(left))
}

func isGoErrorType(value types.Type) bool {
	errorObject := types.Universe.Lookup("error")
	return errorObject != nil && types.Identical(types.Unalias(value), errorObject.Type())
}

func analyzeNativeGoMethodInvocation(selection *types.Selection, receiverAST ast.Expr, argumentsAST []ast.Expr, ellipsis bool, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutableLocals map[types.Object]bool) (*goExpression, error) {
	if selection == nil || selection.Kind() != types.MethodVal || ellipsis {
		return nil, fmt.Errorf("expression.native_method_target")
	}
	function, ok := selection.Obj().(*types.Func)
	if !ok || function.Pkg() == nil {
		return nil, fmt.Errorf("expression.native_method_target")
	}
	declared, ok := function.Type().(*types.Signature)
	if !ok || declared.Recv() == nil {
		return nil, fmt.Errorf("expression.native_method_signature")
	}
	resultType, resultTypes, nativeTypes, ok := nativeGoResultTypeID(declared, records)
	if !ok {
		return nil, fmt.Errorf("expression.native_method_result_type")
	}
	receiver, err := analyzeGoExpressionWithProgram(receiverAST, signature, info, locals, functions, records, mutableLocals)
	if err != nil {
		return nil, err
	}
	arguments := make([]*goExpression, len(argumentsAST))
	for index, argument := range argumentsAST {
		arguments[index], err = analyzeGoExpressionWithProgram(argument, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
	}
	receiverType := types.TypeString(declared.Recv().Type(), func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	})
	target := function.Pkg().Path() + ".(" + receiverType + ")." + function.Name()
	return &goExpression{kind: goNativeMethodInvocation, left: receiver, arguments: arguments, nativeLanguage: "go", nativeTarget: target, nativeSignature: types.TypeString(declared, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	}), nativeResultType: resultType, nativeResultTypes: resultTypes, nativeTypes: nativeTypes}, nil
}

// zeroGoExpression realizes fields omitted from a keyed Go struct literal.
// It uses only existing neutral value constructors; Go nil versus empty
// collection identity is outside this bounded semantic profile.
func zeroGoExpression(value types.Type, records map[*types.Named]goRecordInfo, visiting map[string]bool, depth int) (*goExpression, error) {
	if depth > 32 {
		return nil, fmt.Errorf("expression.record_zero_depth")
	}
	switch {
	case isInt64(value):
		return &goExpression{kind: goIntegerLiteral}, nil
	case isBool(value):
		return &goExpression{kind: goBooleanLiteral}, nil
	case isPureString(value):
		return &goExpression{kind: goStringLiteral}, nil
	case isBytes(value):
		return &goExpression{kind: goBytesLiteral}, nil
	case isPrimitiveSlice(value):
		element, _ := goPrimitiveSliceElement(value)
		tag, _ := goPrimitiveTypeTag(element)
		elementTypeID := goSemanticTypeIdentity(element)
		return &goExpression{kind: goSliceConstruct, typeID: stableID("execution", "type", "slice", tag), elementTypeID: elementTypeID}, nil
	case isI64Map(value):
		return &goExpression{kind: goEmptyMap, typeID: stableID("execution", "type", "map", "i64", "i64")}, nil
	}
	named, ok := types.Unalias(value).(*types.Named)
	if !ok {
		return nil, fmt.Errorf("expression.unsupported_record_zero")
	}
	record, exists := findGoRecord(records, named)
	if !exists || visiting[record.id] {
		return nil, fmt.Errorf("expression.unsupported_record_zero")
	}
	visiting[record.id] = true
	defer delete(visiting, record.id)
	values := make([]*goExpression, len(record.ordered))
	for index, field := range record.ordered {
		fieldValue, err := zeroGoExpression(field.Type(), records, visiting, depth+1)
		if err != nil {
			return nil, err
		}
		values[index] = fieldValue
	}
	return &goExpression{kind: goRecordConstruct, recordType: record.id, values: values}, nil
}

func structuralTaggedMatchCall(function *ast.FuncLit, call *ast.CallExpr, outer *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutable map[types.Object]bool) (*goExpression, bool) {
	if len(call.Args) != 0 || call.Ellipsis.IsValid() || len(function.Body.List) != 3 {
		return nil, false
	}
	bind, ok := function.Body.List[0].(*ast.AssignStmt)
	if !ok || bind.Tok != token.DEFINE || len(bind.Lhs) != 1 || len(bind.Rhs) != 1 {
		return nil, false
	}
	name, ok := bind.Lhs[0].(*ast.Ident)
	if !ok {
		return nil, false
	}
	branch, ok := function.Body.List[1].(*ast.IfStmt)
	if !ok || branch.Init != nil || branch.Else != nil || len(branch.Body.List) != 1 {
		return nil, false
	}
	condition, ok := ast.Unparen(branch.Cond).(*ast.SelectorExpr)
	base, baseOK := condition.X.(*ast.Ident)
	firstReturn, firstOK := branch.Body.List[0].(*ast.ReturnStmt)
	secondReturn, secondOK := function.Body.List[2].(*ast.ReturnStmt)
	if !baseOK || info.Uses[base] != info.Defs[name] || !firstOK || !secondOK || len(firstReturn.Results) != 1 || len(secondReturn.Results) != 1 {
		return nil, false
	}
	value, err := analyzeGoExpressionWithProgram(bind.Rhs[0], outer, info, locals, functions, records, mutable)
	if err != nil {
		return nil, false
	}
	arm := func(node ast.Expr, field string) (*goExpression, bool) {
		if selector, ok := ast.Unparen(node).(*ast.SelectorExpr); ok && selector.Sel.Name == field {
			if identifier, yes := selector.X.(*ast.Ident); yes && info.Uses[identifier] == info.Defs[name] {
				return &goExpression{kind: goVariantRead}, true
			}
		}
		if literal, ok := ast.Unparen(node).(*ast.CompositeLit); ok && len(literal.Elts) == 0 {
			if named, namedOK := types.Unalias(info.TypeOf(literal)).(*types.Named); namedOK {
				if _, recordOK := findGoRecord(records, named); recordOK {
					// Result/option matches represent an absent aggregate arm with
					// the canonical neutral zero placeholder. Projection realizes
					// that placeholder as Go's exact typed zero record.
					return &goExpression{kind: goIntegerLiteral}, true
				}
			}
		}
		result, x := analyzeGoExpressionWithProgram(node, outer, info, locals, functions, records, mutable)
		return result, x == nil
	}
	boundType := info.TypeOf(bind.Rhs[0])
	if boundType == nil {
		return nil, false
	}
	if item, option := goOptionValueType(boundType); option && condition.Sel.Name == "Some" {
		some, a := arm(firstReturn.Results[0], "Value")
		none, b := arm(secondReturn.Results[0], "")
		if !a || !b {
			return nil, false
		}
		return &goExpression{kind: goOptionMatch, left: value, initial: none, body: some, text: "value", typeID: goSemanticTypeIdentity(item)}, true
	}
	if success, failure, result := goResultTypes(boundType); result && condition.Sel.Name == "Ok" {
		good, a := arm(firstReturn.Results[0], "Value")
		bad, b := arm(secondReturn.Results[0], "Error")
		if !a || !b {
			return nil, false
		}
		return &goExpression{kind: goResultMatch, left: value, body: good, alternate: bad, text: "value", typeID: goSemanticTypeIdentity(success), errorName: "failure", errorTypeID: goSemanticTypeIdentity(failure)}, true
	}
	return nil, false
}

func structuralMapLookupOptionCall(function *ast.FuncLit, call *ast.CallExpr, outer *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutable map[types.Object]bool) (*goExpression, bool) {
	if len(call.Args) != 0 || call.Ellipsis.IsValid() || len(function.Body.List) != 2 {
		return nil, false
	}
	signature, ok := info.TypeOf(function.Type).(*types.Signature)
	if !ok || signature.Params().Len() != 0 || signature.Results().Len() != 1 {
		return nil, false
	}
	item, ok := goOptionValueType(signature.Results().At(0).Type())
	if !ok || !isInt64(item) {
		return nil, false
	}
	bind, ok := function.Body.List[0].(*ast.AssignStmt)
	if !ok || bind.Tok != token.DEFINE || len(bind.Lhs) != 2 || len(bind.Rhs) != 1 {
		return nil, false
	}
	valueName, valueOK := bind.Lhs[0].(*ast.Ident)
	foundName, foundOK := bind.Lhs[1].(*ast.Ident)
	lookup, lookupOK := ast.Unparen(bind.Rhs[0]).(*ast.IndexExpr)
	returned, returnOK := function.Body.List[1].(*ast.ReturnStmt)
	if !valueOK || !foundOK || !lookupOK || !returnOK || len(returned.Results) != 1 {
		return nil, false
	}
	literal, literalOK := ast.Unparen(returned.Results[0]).(*ast.CompositeLit)
	if !literalOK || !types.Identical(info.TypeOf(literal), signature.Results().At(0).Type()) {
		return nil, false
	}
	fields, err := keyedCompositeFields(literal)
	if err != nil || len(fields) != 2 {
		return nil, false
	}
	some, someOK := fields["Some"].(*ast.Ident)
	value, fieldOK := fields["Value"].(*ast.Ident)
	if !someOK || !fieldOK || info.Uses[some] != info.Defs[foundName] || info.Uses[value] != info.Defs[valueName] {
		return nil, false
	}
	mapping, err := analyzeGoExpressionWithProgram(lookup.X, outer, info, locals, functions, records, mutable)
	if err != nil {
		return nil, false
	}
	key, err := analyzeGoExpressionWithProgram(lookup.Index, outer, info, locals, functions, records, mutable)
	if err != nil {
		return nil, false
	}
	return &goExpression{kind: goMapLookupOption, left: mapping, right: key, typeID: goSemanticTypeIdentity(signature.Results().At(0).Type())}, true
}

func findGoRecord(records map[*types.Named]goRecordInfo, named *types.Named) (goRecordInfo, bool) {
	if named == nil {
		return goRecordInfo{}, false
	}
	if record, ok := records[named]; ok {
		return record, true
	}
	origin := named.Origin()
	if record, ok := records[origin]; ok {
		return record, true
	}
	for candidate, record := range records {
		if sameNamedType(candidate, origin) {
			return record, true
		}
	}
	return goRecordInfo{}, false
}

func sameNamedType(left, right *types.Named) bool {
	if left == nil || right == nil || left.Obj() == nil || right.Obj() == nil || left.Obj().Name() != right.Obj().Name() {
		return false
	}
	leftPackage, rightPackage := left.Obj().Pkg(), right.Obj().Pkg()
	return leftPackage == nil && rightPackage == nil || leftPackage != nil && rightPackage != nil && leftPackage.Path() == rightPackage.Path()
}

func isSlicesFunction(info *types.Info, selector *ast.SelectorExpr, name string) bool {
	function, ok := info.Uses[selector.Sel].(*types.Func)
	return ok && function.Name() == name && function.Pkg() != nil && function.Pkg().Path() == "slices"
}

func structuralImmutableMapCall(function *ast.FuncLit, call *ast.CallExpr, info *types.Info) (goExpressionKind, bool) {
	signature, ok := info.TypeOf(function).(*types.Signature)
	if !ok || signature.Results().Len() != 1 || !isI64Map(signature.Results().At(0).Type()) || signature.Params().Len() < 2 || signature.Params().Len() > 3 || len(call.Args) != signature.Params().Len() || call.Ellipsis.IsValid() {
		return 0, false
	}
	if !isI64Map(signature.Params().At(0).Type()) || !isInt64(signature.Params().At(1).Type()) || (signature.Params().Len() == 3 && !isInt64(signature.Params().At(2).Type())) || len(function.Body.List) != 3 {
		return 0, false
	}
	clone, ok := function.Body.List[0].(*ast.AssignStmt)
	if !ok || clone.Tok != token.DEFINE || len(clone.Lhs) != 1 || len(clone.Rhs) != 1 {
		return 0, false
	}
	out, ok := clone.Lhs[0].(*ast.Ident)
	if !ok {
		return 0, false
	}
	cloneCall, ok := ast.Unparen(clone.Rhs[0]).(*ast.CallExpr)
	if !ok || len(cloneCall.Args) != 1 {
		return 0, false
	}
	cloneSelector, ok := ast.Unparen(cloneCall.Fun).(*ast.SelectorExpr)
	if !ok || !isMapsFunction(info, cloneSelector, "Clone") {
		return 0, false
	}
	m, ok := ast.Unparen(cloneCall.Args[0]).(*ast.Ident)
	if !ok || info.Uses[m] != signature.Params().At(0) {
		return 0, false
	}
	ret, ok := function.Body.List[2].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return 0, false
	}
	retOut, ok := ast.Unparen(ret.Results[0]).(*ast.Ident)
	if !ok || info.Uses[retOut] != info.Defs[out] {
		return 0, false
	}
	if signature.Params().Len() == 3 {
		assign, ok := function.Body.List[1].(*ast.AssignStmt)
		if !ok || assign.Tok != token.ASSIGN || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return 0, false
		}
		index, ok := ast.Unparen(assign.Lhs[0]).(*ast.IndexExpr)
		if !ok {
			return 0, false
		}
		x, xok := ast.Unparen(index.X).(*ast.Ident)
		k, kok := ast.Unparen(index.Index).(*ast.Ident)
		v, vok := ast.Unparen(assign.Rhs[0]).(*ast.Ident)
		if !xok || !kok || !vok || info.Uses[x] != info.Defs[out] || info.Uses[k] != signature.Params().At(1) || info.Uses[v] != signature.Params().At(2) {
			return 0, false
		}
		return goMapUpdate, true
	}
	statement, ok := function.Body.List[1].(*ast.ExprStmt)
	if !ok {
		return 0, false
	}
	remove, ok := ast.Unparen(statement.X).(*ast.CallExpr)
	if !ok || len(remove.Args) != 2 {
		return 0, false
	}
	deleteID, ok := ast.Unparen(remove.Fun).(*ast.Ident)
	if !ok || info.Uses[deleteID] != types.Universe.Lookup("delete") {
		return 0, false
	}
	x, xok := ast.Unparen(remove.Args[0]).(*ast.Ident)
	k, kok := ast.Unparen(remove.Args[1]).(*ast.Ident)
	if !xok || !kok || info.Uses[x] != info.Defs[out] || info.Uses[k] != signature.Params().At(1) {
		return 0, false
	}
	return goMapRemove, true
}

func isMapsFunction(info *types.Info, selector *ast.SelectorExpr, name string) bool {
	function, ok := info.Uses[selector.Sel].(*types.Func)
	return ok && function.Name() == name && function.Pkg() != nil && function.Pkg().Path() == "maps"
}

func isBytesFunction(info *types.Info, selector *ast.SelectorExpr, name string) bool {
	function, ok := info.Uses[selector.Sel].(*types.Func)
	return ok && function.Name() == name && function.Pkg() != nil && function.Pkg().Path() == "bytes"
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

func goI64IndexExpressionKey(expression ast.Expr, info *types.Info) (string, bool) {
	expression = ast.Unparen(expression)
	if conversion, ok := expression.(*ast.CallExpr); ok && len(conversion.Args) == 1 {
		if typeName, yes := ast.Unparen(conversion.Fun).(*ast.Ident); yes && typeName.Name == "int" && info.Uses[typeName] == types.Universe.Lookup("int") {
			return goI64IndexExpressionKey(conversion.Args[0], info)
		}
	}
	switch value := expression.(type) {
	case *ast.Ident:
		object := info.Uses[value]
		return fmt.Sprintf("object:%p", object), object != nil && isInt64(info.TypeOf(value))
	case *ast.SelectorExpr:
		selection := info.Selections[value]
		if selection == nil {
			return "", false
		}
		base, ok := goI64IndexExpressionBaseKey(value.X, info)
		return base + "." + selection.Obj().Id(), ok && isInt64(info.TypeOf(value))
	default:
		return "", false
	}
}

func goI64IndexExpressionBaseKey(expression ast.Expr, info *types.Info) (string, bool) {
	if identifier, ok := ast.Unparen(expression).(*ast.Ident); ok {
		object := info.Uses[identifier]
		return fmt.Sprintf("object:%p", object), object != nil
	}
	return "", false
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

func isGoIntegerExpression(expression ast.Expr, info *types.Info) bool {
	typeOf := info.TypeOf(expression)
	if typeOf == nil {
		return false
	}
	basic, ok := typeOf.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsInteger != 0
}

// Rewriting integer inequality through ordering would duplicate evaluation of
// each operand. Until canonical let-binding is available, accept only stable
// source operands for which that duplication is observationally exact.
func stableIntegerComparisonOperand(expression ast.Expr, info *types.Info) bool {
	switch value := ast.Unparen(expression).(type) {
	case *ast.BasicLit, *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return stableIntegerComparisonOperand(value.X, info)
	case *ast.IndexExpr:
		return stableIntegerComparisonOperand(value.X, info) && stableIntegerComparisonOperand(value.Index, info)
	case *ast.BinaryExpr:
		return (value.Op == token.ADD || value.Op == token.SUB || value.Op == token.MUL) && stableIntegerComparisonOperand(value.X, info) && stableIntegerComparisonOperand(value.Y, info)
	case *ast.CallExpr:
		identifier, ok := ast.Unparen(value.Fun).(*ast.Ident)
		if !ok || len(value.Args) != 1 {
			return false
		}
		if identifier.Name == "len" && info.Uses[identifier] == types.Universe.Lookup("len") {
			return stableIntegerComparisonOperand(value.Args[0], info)
		}
		if identifier.Name == "int64" && info.Uses[identifier] == types.Universe.Lookup("int64") {
			return stableIntegerComparisonOperand(value.Args[0], info)
		}
		return false
	default:
		return false
	}
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
	case goBooleanNot:
		value, err := evaluateBooleanExpression(expression.left, parameters)
		return !value, err
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
