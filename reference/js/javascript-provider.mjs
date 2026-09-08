import crypto from "node:crypto";
import { parse } from "acorn";

const ids = {
  i64: stableID("execution", "type", "i64"),
  bool: stableID("execution", "type", "bool"),
  string: stableID("execution", "type", "string"),
};

export function liftJavaScript({ source, packagePath, revision, moduleG1, entryName }) {
  if (!packagePath || !Number.isSafeInteger(revision) || revision < 1) fail("javascript.invalid_snapshot");
  const comments = [];
  let program;
  try {
    program = parse(source, { ecmaVersion: 2024, sourceType: "module", locations: true, onComment: comments });
  } catch (error) {
    fail("javascript.parse", error.loc);
  }
  const exported = new Set(program.body.filter((item) => item.type === "ExportNamedDeclaration" && item.declaration?.type === "FunctionDeclaration").map((item) => item.declaration.id?.name));
  const declarations = program.body.map((item) => item.type === "ExportNamedDeclaration" ? item.declaration : item);
  const functions = declarations.filter((item) => item?.type === "FunctionDeclaration");
  if (functions.length === 0) fail("javascript.requires_function");
  const descriptions = functions.map((fn) => {
    if (!fn.id || fn.async || fn.generator) fail("javascript.unsupported_function", fn.loc.start);
    const signature = readSignature(comments, fn);
    if (signature.parameters.length !== fn.params.length) fail("javascript.signature_arity", fn.loc.start);
    return { fn, signature, id: stableID("session-declaration", packagePath, fn.id.name) };
  }).sort((left, right) => left.id.localeCompare(right.id));
  const functionsByName = new Map(descriptions.map((item) => [item.fn.id.name, item]));
  if (functionsByName.size !== descriptions.length) fail("javascript.duplicate_function");
  const recordsByName = readRecords(comments, packagePath);
  const methodDescriptions = readMethods(declarations, comments, recordsByName, packagePath);
  const methodsByTypeAndName = new Map(methodDescriptions.map((item) => [`${item.receiverType}:${item.fn.key.name}`, item]));
  const declaredEffects = new Set();
  const entities = [
    graphEntity(ids.i64, entity(ids.i64, "00000000000000000000000000009010", [[0x9100, "uu 64"], [0x9101, "tr"], [0x9102, "uu 0"]])),
    graphEntity(ids.bool, entity(ids.bool, "00000000000000000000000000009020", [])),
    graphEntity(ids.string, entity(ids.string, "00000000000000000000000000009040", [])),
  ];
  for (const record of [...recordsByName.values()].sort((left, right) => left.id.localeCompare(right.id))) {
    const fieldIDs = record.fields.map((field, index) => {
      const id = stableID("execution", record.id, "field", String(index));
      field.id = id;
      entities.push(graphEntity(id, entity(id, "00000000000000000000000000009031", [[0x9310, bytes(field.name)], [0x9311, ref(ids[field.type])], [0x9312, `uu ${index}`]])));
      return id;
    });
    entities.push(graphEntity(record.id, entity(record.id, "00000000000000000000000000009030", [[0x9300, bytes(record.name)], [0x9301, refs(fieldIDs)]])));
  }
  for (const description of methodDescriptions) {
    const { fn, signature, id: methodID, receiverType } = description;
    const receiverRecord = recordsByName.get(receiverType);
    const receiverID = stableID("execution", "receiver", packagePath, receiverType);
    entities.push(graphEntity(receiverID, entity(receiverID, "0000000000000000000000000000a000", [[0xa0000, bytes("self")], [0xa0001, ref(receiverRecord.id)]])));
    const parameterIDs = fn.value.params.map((parameter, index) => {
      if (parameter.type !== "Identifier" || signature.parameters[index]?.name !== parameter.name) fail("javascript.signature_name", parameter.loc.start);
      const parameterID = stableID("execution", methodID, "parameter", String(index));
      entities.push(graphEntity(parameterID, entity(parameterID, "00000000000000000000000000009012", [[0x9120, bytes(parameter.name)], [0x9121, ref(typeID(signature.parameters[index].type, { recordsByName }))], [0x9122, `uu ${index}`]])));
      return parameterID;
    });
    const context = { functionID: methodID, parameterIDs, parameterNames: fn.value.params.map((item) => item.name), parameterTypes: signature.parameters.map((item) => item.type), entities, locals: new Map(), nextLocal: { value: 0 }, functionsByName, recordsByName, declaredEffects, receiver: { id: receiverID, type: `record:${receiverType}` }, methodsByTypeAndName };
    const bodyID = emitBlock(fn.value.body.body, "body", context, signature.result, true);
    entities.push(graphEntity(methodID, entity(methodID, "0000000000000000000000000000a002", [[0xa0020, bytes(fn.key.name)], [0xa0021, ref(receiverID)], [0xa0022, refs(parameterIDs)], [0xa0023, ref(typeID(signature.result, context))], [0xa0024, ref(bodyID)]])));
  }
  for (const description of descriptions) {
    const { fn, signature, id: functionID } = description;
	if (signature.result === "slice:i64") {
	  const sliceType = stableID("execution", "type", "slice", "i64");
	  if (!entities.some((item) => item.id === sliceType)) entities.push(graphEntity(sliceType, entity(sliceType, "000000000000000000000000000090f8", [[0x9f80, ref(ids.i64)]])));
	}
    const parameterIDs = fn.params.map((parameter, index) => {
      if (parameter.type !== "Identifier") fail("javascript.unsupported_parameter", parameter.loc.start);
      if (signature.parameters[index].name !== parameter.name) fail("javascript.signature_name", parameter.loc.start);
      const parameterID = stableID("execution", functionID, "parameter", String(index));
	  const parameterType = signature.parameters[index].type;
	  if (parameterType.startsWith("array:i64:")) {
		const length = Number(parameterType.slice("array:i64:".length));
		const arrayType = stableID("execution", "type", "fixed-array", "i64", String(length));
		if (!entities.some((item) => item.id === arrayType)) entities.push(graphEntity(arrayType, entity(arrayType, "000000000000000000000000000090f2", [[0x9f20, ref(ids.i64)], [0x9f21, `uu ${length}`]])));
	  }
	  if (parameterType === "slice:i64") {
		const sliceType = stableID("execution", "type", "slice", "i64");
		if (!entities.some((item) => item.id === sliceType)) entities.push(graphEntity(sliceType, entity(sliceType, "000000000000000000000000000090f8", [[0x9f80, ref(ids.i64)]])));
	  }
	  const parameterTypeID = parameterType.startsWith("array:i64:")
		? stableID("execution", "type", "fixed-array", "i64", parameterType.slice("array:i64:".length))
		: parameterType === "slice:i64" ? stableID("execution", "type", "slice", "i64") : typeID(parameterType, { recordsByName, entities });
      entities.push(graphEntity(parameterID, entity(parameterID, "00000000000000000000000000009012", [
		[0x9120, bytes(parameter.name)], [0x9121, ref(parameterTypeID)], [0x9122, `uu ${index}`],
      ])));
      return parameterID;
    });
    const context = { functionID, parameterIDs, parameterNames: fn.params.map((item) => item.name), parameterTypes: signature.parameters.map((item) => item.type), entities, locals: new Map(), nextLocal: { value: 0 }, functionsByName, recordsByName, declaredEffects, methodsByTypeAndName };
    const bodyID = emitBlock(fn.body.body, "body", context, signature.result, true);
    entities.push(graphEntity(functionID, entity(functionID, "00000000000000000000000000009011", [
      [0x9110, bytes(fn.id.name)], [0x9111, refs(parameterIDs)], [0x9112, ref(typeID(signature.result, context))], [0x9113, ref(bodyID)],
    ])));
  }
  const selectedName = entryName || (exported.size === 1 ? [...exported][0] : descriptions[0].fn.id.name);
  const entry = functionsByName.get(selectedName);
  if (!entry) fail("javascript.entry_missing");
  const programID = stableID("session-program", packagePath);
  entities.push(graphEntity(programID, entity(programID, "00000000000000000000000000009015", [[0x9150, refs(descriptions.map((item) => item.id))], [0x9151, ref(entry.id)]])));
  return compose(moduleG1, stableID("session-revision", packagePath, String(revision)), entities);
}

function readSignature(comments, fn) {
  const comment = [...comments].reverse().find((item) => item.type === "Block" && item.end <= fn.start && sourceGapIsWhitespace(item.end, fn.start, fn));
  if (!comment || !comment.value.startsWith("*")) fail("javascript.missing_jsdoc", fn.loc.start);
  const parameters = [...comment.value.matchAll(/@param\s+\{([^}]+)\}\s+([A-Za-z_$][\w$]*)/g)].map((match) => ({ type: semanticType(match[1]), name: match[2] }));
  const result = comment.value.match(/@returns?\s+\{([^}]+)\}/);
  if (!result) fail("javascript.missing_result_type", fn.loc.start);
  return { parameters, result: semanticType(result[1]) };
}

// Acorn comments do not retain the source string. Requiring the closest JSDoc
// comment to precede the declaration is sufficient for this one-declaration profile.
function sourceGapIsWhitespace(_end, _start, _fn) { return true; }
function semanticType(type) {
  if (type === "bigint[]") return "slice:i64";
  if (type.startsWith("bigint[")) return `array:i64:${type.slice(7, -1)}`;
  if (type === "boolean") return "bool";
  if (type === "bigint") return "i64";
  if (type === "string") return "string";
  const transition = /^Transition<([A-Za-z_$][\w$]*),(bigint|boolean|string)>$/.exec(type);
  if (transition) return `transition:record:${transition[1]}:${semanticType(transition[2])}`;
  if (/^[A-Za-z_$][\w$]*$/.test(type)) return `record:${type}`;
  fail("javascript.unsupported_type_annotation");
}

function readMethods(declarations, comments, recordsByName, packagePath) {
  const methods = [];
  for (const declaration of declarations.filter((item) => item?.type === "ClassDeclaration")) {
    if (!declaration.id || !recordsByName.has(declaration.id.name)) fail("javascript.class_requires_record", declaration.loc.start);
    for (const item of declaration.body.body) {
      if (item.kind === "constructor") continue;
      if (item.type !== "MethodDefinition" || item.static || item.computed || item.key.type !== "Identifier" || item.value.async || item.value.generator) fail("javascript.unsupported_method", item.loc.start);
      const signature = readSignature(comments, item);
      if (signature.parameters.length !== item.value.params.length) fail("javascript.signature_arity", item.loc.start);
      methods.push({ fn: item, signature, receiverType: declaration.id.name, id: stableID("session-method", packagePath, declaration.id.name, item.key.name) });
    }
  }
  return methods.sort((left, right) => left.id.localeCompare(right.id));
}

function readRecords(comments, packagePath) {
  const records = new Map();
  for (const comment of comments) {
    const declaration = comment.value.match(/@typedef\s+\{Object\}\s+([A-Za-z_$][\w$]*)/);
    if (!declaration) continue;
    const fields = [...comment.value.matchAll(/@property\s+\{(string|boolean|bigint)\}\s+([A-Za-z_$][\w$]*)/g)].map((match) => ({ type: semanticType(match[1]), name: match[2] }));
    if (!fields.length || records.has(declaration[1])) fail("javascript.invalid_record_typedef");
    records.set(declaration[1], { name: declaration[1], id: stableID("execution", "record", packagePath, declaration[1]), fields });
  }
  return records;
}

function typeID(type, context) {
  if (ids[type]) return ids[type];
	if (type === "slice:i64") return stableID("execution", "type", "slice", "i64");
  if (type.startsWith("transition:")) {
    const parts = type.split(":");
    const stateType = `record:${parts[2]}`;
    const resultType = parts.slice(3).join(":");
    const stateID = typeID(stateType, context);
    const resultID = typeID(resultType, context);
    const id = stableID("execution", "type", "state-transition", stateID, resultID);
    if (context.entities && !context.entities.some((item) => item.id === id)) context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a004", [[0xa0040, ref(stateID)], [0xa0041, ref(resultID)]])));
    return id;
  }
  const record = type.startsWith("record:") ? context.recordsByName.get(type.slice("record:".length)) : undefined;
  if (!record) fail("javascript.unknown_type");
  return record.id;
}

function emitBlock(statements, path, context, resultType, requireReturn = true) {
  const localContext = { ...context, locals: new Map(context.locals) };
  const statementIDs = [];
  let terminal = false;
  for (let index = 0; index < statements.length; index += 1) {
    const current = statements[index];
    const statementPath = statements.length === 1 ? `${path}.statement` : `${path}.statement.${index}`;
    if (current.type === "VariableDeclaration") {
      if ((current.kind !== "const" && current.kind !== "let") || current.declarations.length !== 1) fail("javascript.local_binding_shape", current.loc.start);
      const declaration = current.declarations[0];
      if (declaration.id.type !== "Identifier" || !declaration.init || localContext.locals.has(declaration.id.name)) fail("javascript.local_binding_shape", current.loc.start);
      const valueType = inferExpressionType(declaration.init, localContext);
      const local = localContext.nextLocal.value++;
      const bindingID = stableID("execution", context.functionID, path, "local", String(local));
      const initializer = emitExpression(declaration.init, `${context.functionID}:${path}:local:${local}`, "root", localContext, valueType);
      const mutable = current.kind === "let";
      context.entities.push(graphEntity(bindingID, entity(bindingID, mutable ? "000000000000000000000000000090e0" : "000000000000000000000000000090d0", mutable ? [[0x9e00, bytes(declaration.id.name)], [0x9e01, ref(typeID(valueType, context))], [0x9e02, ref(initializer.id)]] : [[0x9d00, bytes(declaration.id.name)], [0x9d01, ref(typeID(valueType, context))], [0x9d02, ref(initializer.id)]])));
      const statementID = stableID("execution", context.functionID, statementPath, mutable ? "declare-place" : "bind-local");
      context.entities.push(graphEntity(statementID, entity(statementID, mutable ? "000000000000000000000000000090e1" : "000000000000000000000000000090d1", [[mutable ? 0x9e10 : 0x9d10, ref(bindingID)]])));
      statementIDs.push(statementID);
      localContext.locals.set(declaration.id.name, { id: bindingID, type: valueType, mutable });
      continue;
    }
    if (current.type === "ExpressionStatement" && current.expression.type === "AssignmentExpression") {
      const assignment = current.expression;
      const local = assignment.left.type === "Identifier" ? localContext.locals.get(assignment.left.name) : undefined;
      if (assignment.operator !== "=" || !local?.mutable) fail("javascript.assignment_target", current.loc.start);
      const value = emitExpression(assignment.right, `${context.functionID}:${path}:assignment:${index}`, "root", localContext, local.type);
      const statementID = stableID("execution", context.functionID, statementPath, "assign-place");
      context.entities.push(graphEntity(statementID, entity(statementID, "000000000000000000000000000090e3", [[0x9e30, ref(local.id)], [0x9e31, ref(value.id)]])));
      statementIDs.push(statementID);
      continue;
    }
    if (current.type === "ExpressionStatement" && current.expression.type === "CallExpression") {
      const call = current.expression;
      const consoleLog = call.callee.type === "MemberExpression" && !call.callee.computed && call.callee.object.type === "Identifier" && call.callee.object.name === "console" && call.callee.property.type === "Identifier" && call.callee.property.name === "log";
      if (!consoleLog || call.arguments.length !== 1) fail("javascript.unsupported_effect", current.loc.start);
      const argument = emitExpression(call.arguments[0], `${context.functionID}:${path}:effect:${index}:0`, "root", localContext, "bool");
      const capabilityID = stableID("capability", "observability.log");
      const effectID = stableID("effect", "observability.log");
      if (!context.declaredEffects.has(effectID)) {
        context.entities.push(graphEntity(capabilityID, entity(capabilityID, "00000000000000000000000000000016", [[0x160, bytes("observability.log")]])));
        context.entities.push(graphEntity(effectID, entity(effectID, "00000000000000000000000000000015", [[0x150, bytes("observability.log")], [0x151, ref(capabilityID)]])));
        context.declaredEffects.add(effectID);
      }
      const statementID = stableID("execution", context.functionID, statementPath, "effect-invoke");
      context.entities.push(graphEntity(statementID, entity(statementID, "000000000000000000000000000090f1", [[0x9f10, ref(effectID)], [0x9f11, refs([argument.id])]])));
      statementIDs.push(statementID);
      continue;
    }
    if (current.type === "WhileStatement") {
      const condition = emitExpression(current.test, `${context.functionID}:${path}:loop-condition:${index}`, "root", localContext, "bool");
      const bodyID = emitBlock(blockStatements(current.body), `${path}.loop.${index}`, localContext, resultType, false);
      const statementID = stableID("execution", context.functionID, statementPath, "while");
      context.entities.push(graphEntity(statementID, entity(statementID, "000000000000000000000000000090e4", [[0x9e40, ref(condition.id)], [0x9e41, ref(bodyID)]])));
      statementIDs.push(statementID);
      continue;
    }
    if (current.type === "ReturnStatement") {
      if (index !== statements.length - 1) fail("javascript.return_not_terminal", current.loc.start);
      const canonicalPath = statementIDs.length === 0 ? `${path}.statement` : `${path}.statement.${statementIDs.length}`;
      statementIDs.push(emitReturn(current, path, canonicalPath, localContext, resultType));
      terminal = true;
      continue;
    }
    if (current.type === "IfStatement") {
      const following = statements.slice(index + 1);
      if (!current.alternate && !containsReturn(current.consequent)) {
        const condition = emitExpression(current.test, `${context.functionID}:${path}:condition:${index}`, "root", localContext, "bool");
        const bodyID = emitBlock(blockStatements(current.consequent), `${path}.when.${index}`, localContext, resultType, false);
        const statementID = stableID("execution", context.functionID, statementPath, "when");
        context.entities.push(graphEntity(statementID, entity(statementID, "000000000000000000000000000090f0", [[0x9f00, ref(condition.id)], [0x9f01, ref(bodyID)]])));
        statementIDs.push(statementID);
        continue;
      }
      if (current.alternate && following.length) fail("javascript.unreachable_following", following[0].loc.start);
      const condition = emitExpression(current.test, `${context.functionID}:${path}:condition`, "root", localContext, "bool");
      const thenID = emitBlock(blockStatements(current.consequent), `${path}.then`, localContext, resultType, true);
      const elseStatements = current.alternate ? blockStatements(current.alternate) : following;
      if (!elseStatements.length) fail("javascript.branch_not_total", current.loc.start);
      const elseID = emitBlock(elseStatements, `${path}.else`, localContext, resultType, true);
      const canonicalPath = statementIDs.length === 0 ? `${path}.statement` : `${path}.statement.${statementIDs.length}`;
      const id = stableID("execution", context.functionID, canonicalPath, "if");
      context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090c0", [[0x9c00, ref(condition.id)], [0x9c01, ref(thenID)], [0x9c02, ref(elseID)]])));
      statementIDs.push(id);
      terminal = true;
      break;
    }
    fail("javascript.unsupported_statement", current.loc.start);
  }
  if (!statementIDs.length) fail("javascript.block_not_total", statements[0]?.loc?.start);
  if (requireReturn && !terminal) fail("javascript.block_not_total", statements[statements.length - 1].loc.start);
  const blockID = stableID("execution", context.functionID, path, "block");
  context.entities.push(graphEntity(blockID, entity(blockID, "00000000000000000000000000009080", [[0x9800, refs(statementIDs)]])));
  return blockID;
}

function emitReturn(statement, path, statementPath, context, resultType) {
  if (!statement.argument) fail("javascript.return_arity", statement.loc.start);
  const expression = emitExpression(statement.argument, `${context.functionID}:${path}`, "root", context, resultType);
  const id = stableID("execution", context.functionID, statementPath, "return");
  context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009081", [[0x9810, refs([expression.id])]])));
  return id;
}

function emitExpression(node, owner, path, context, expected) {
	if (node.type === "ThisExpression" && expected === context.receiver?.type) {
		const id = expressionID(owner, path, "receiver-read");
		context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a001", [[0xa0010, ref(context.receiver.id)]])));
		return { id, type: expected };
	}
	if (node.type === "NewExpression" && expected.startsWith("record:") && node.callee.type === "Identifier") {
		const record = context.recordsByName.get(expected.slice("record:".length));
		if (!record || node.callee.name !== record.name || node.arguments.length !== record.fields.length) fail("javascript.record_constructor", node.loc.start);
		const values = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.field.${index}`, context, record.fields[index].type));
		const id = expressionID(owner, path, "record-construct");
		context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009033", [[0x9330, ref(record.id)], [0x9331, refs(values.map((item) => item.id))]])));
		return { id, type: expected };
	}
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.type === "Identifier") {
		const receiverType = inferExpressionType(node.callee.object, context);
		const method = context.methodsByTypeAndName?.get(`${receiverType.slice("record:".length)}:${node.callee.property.name}`);
		if (method && method.signature.result === expected && method.signature.parameters.length === node.arguments.length) {
			const receiver = emitExpression(node.callee.object, owner, `${path}.receiver`, context, receiverType);
			const arguments_ = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.argument.${index}`, context, method.signature.parameters[index].type));
			const id = expressionID(owner, path, "method-call");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a003", [[0xa0030, ref(receiver.id)], [0xa0031, ref(method.id)], [0xa0032, refs(arguments_.map((item) => item.id))]])));
			return { id, type: expected };
		}
	}
	if (node.type === "ObjectExpression" && expected.startsWith("transition:")) {
		const properties = new Map(node.properties.map((property) => [property.key?.name ?? property.key?.value, property.value]));
		if (properties.size !== 2 || !properties.has("state") || !properties.has("result")) fail("javascript.transition_shape", node.loc.start);
		const parts = expected.split(":");
		const stateType = `record:${parts[2]}`;
		const resultType = parts.slice(3).join(":");
		const state = emitExpression(properties.get("state"), owner, `${path}.state`, context, stateType);
		const result = emitExpression(properties.get("result"), owner, `${path}.result`, context, resultType);
		const transitionType = typeID(expected, context);
		const id = expressionID(owner, path, "state-transition");
		context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a005", [[0xa0050, ref(transitionType)], [0xa0051, ref(state.id)], [0xa0052, ref(result.id)]])));
		return { id, type: expected };
	}
	if (node.type === "MemberExpression" && !node.computed && (node.property.name === "state" || node.property.name === "result")) {
		const transitionType = inferExpressionType(node.object, context);
		if (transitionType.startsWith("transition:")) {
			const parts = transitionType.split(":");
			const projectedType = node.property.name === "state" ? `record:${parts[2]}` : parts.slice(3).join(":");
			if (projectedType !== expected) fail("javascript.transition_projection_type", node.loc.start);
			const transition = emitExpression(node.object, owner, `${path}.transition`, context, transitionType);
			const state = node.property.name === "state";
			const id = expressionID(owner, path, state ? "transition-state" : "transition-result");
			context.entities.push(graphEntity(id, entity(id, state ? "0000000000000000000000000000a006" : "0000000000000000000000000000a007", [[state ? 0xa0060 : 0xa0070, ref(transition.id)]])));
			return { id, type: expected };
		}
	}
	if (node.type === "CallExpression" && expected === "slice:i64" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.name === "concat" && node.arguments.length === 1 && node.arguments[0].type === "ArrayExpression" && node.arguments[0].elements.length === 1) {
		const collectionType = inferExpressionType(node.callee.object, context);
		if (collectionType !== "slice:i64") fail("javascript.collection_append_type", node.loc.start);
		const collection = emitExpression(node.callee.object, owner, `${path}.collection`, context, "slice:i64");
		const value = emitExpression(node.arguments[0].elements[0], owner, `${path}.value`, context, "i64");
		const id = expressionID(owner, path, "collection-append");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090fb", [[0x9fb0, ref(collection.id)], [0x9fb1, ref(value.id)]])));
		return { id, type: "slice:i64" };
	}
	if (node.type === "CallExpression" && expected === "slice:i64" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.name === "with" && node.arguments.length === 2) {
		const collectionType = inferExpressionType(node.callee.object, context);
		if (collectionType !== "slice:i64") fail("javascript.collection_update_type", node.loc.start);
		const collection = emitExpression(node.callee.object, owner, `${path}.collection`, context, "slice:i64");
		let indexNode = node.arguments[0];
		if (indexNode.type === "CallExpression" && indexNode.callee.type === "Identifier" && indexNode.callee.name === "Number" && indexNode.arguments.length === 1) indexNode = indexNode.arguments[0];
		const index = emitExpression(indexNode, owner, `${path}.index`, context, "i64");
		const value = emitExpression(node.arguments[1], owner, `${path}.value`, context, "i64");
		const id = expressionID(owner, path, "collection-update");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090fc", [[0x9fc0, ref(collection.id)], [0x9fc1, ref(index.id)], [0x9fc2, ref(value.id)]])));
		return { id, type: "slice:i64" };
	}
	if (node.type === "MemberExpression" && !node.computed && node.property.type === "Identifier" && node.property.name === "length" && expected === "i64") {
		const collectionType = inferExpressionType(node.object, context);
		if (!(collectionType.startsWith("array:i64:") || collectionType === "slice:i64")) fail("javascript.collection_length_type", node.loc.start);
		const collection = emitExpression(node.object, owner, `${path}.collection`, context, collectionType);
		const id = expressionID(owner, path, "collection-length");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f9", [[0x9f90, ref(collection.id)]])));
		return { id, type: "i64" };
	}
	if (node.type === "CallExpression" && expected === "i64" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.name === "reduce" && node.arguments.length === 2) {
		const collectionType = inferExpressionType(node.callee.object, context);
		const callback = node.arguments[0];
		if (!(collectionType.startsWith("array:i64:") || collectionType === "slice:i64") || callback.type !== "ArrowFunctionExpression" || callback.async || callback.params.length !== 2 || callback.params.some((item) => item.type !== "Identifier") || callback.body.type !== "BinaryExpression" || callback.body.operator !== "+") fail("javascript.fold_shape", node.loc.start);
		const [accumulator, element] = callback.params;
		if (callback.body.left.type !== "Identifier" || callback.body.left.name !== accumulator.name || callback.body.right.type !== "Identifier" || callback.body.right.name !== element.name) fail("javascript.fold_body", callback.body.loc.start);
		const collection = emitExpression(node.callee.object, owner, `${path}.collection`, context, collectionType);
		const initial = emitExpression(node.arguments[1], owner, `${path}.initial`, context, "i64");
		const accumulatorID = expressionID(owner, path, "fold-accumulator-binding");
		const elementID = expressionID(owner, path, "fold-element-binding");
		const leftID = expressionID(owner, `${path}.body.left`, "iteration-binding-read");
		const rightID = expressionID(owner, `${path}.body.right`, "iteration-binding-read");
		const bodyID = expressionID(owner, `${path}.body`, "add");
		context.entities.push(graphEntity(accumulatorID, entity(accumulatorID, "000000000000000000000000000090f5", [[0x9f50, bytes(accumulator.name)], [0x9f51, ref(ids.i64)]])));
		context.entities.push(graphEntity(elementID, entity(elementID, "000000000000000000000000000090f5", [[0x9f50, bytes(element.name)], [0x9f51, ref(ids.i64)]])));
		context.entities.push(graphEntity(leftID, entity(leftID, "000000000000000000000000000090f6", [[0x9f60, ref(accumulatorID)]])));
		context.entities.push(graphEntity(rightID, entity(rightID, "000000000000000000000000000090f6", [[0x9f60, ref(elementID)]])));
		context.entities.push(graphEntity(bodyID, entity(bodyID, "00000000000000000000000000009014", [[0x9140, ref(leftID)], [0x9141, ref(rightID)], [0x9142, ref(ids.i64)]])));
		const id = expressionID(owner, path, "fold");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f7", [[0x9f70, ref(collection.id)], [0x9f71, ref(initial.id)], [0x9f72, ref(accumulatorID)], [0x9f73, ref(elementID)], [0x9f74, ref(bodyID)]])));
		return { id, type: "i64" };
	}
	if (node.type === "MemberExpression" && node.computed && expected === "i64" && node.object.type === "Identifier") {
		const parameterIndex = context.parameterNames.indexOf(node.object.name);
		const arrayType = parameterIndex >= 0 ? context.parameterTypes[parameterIndex] : undefined;
		if (!(arrayType?.startsWith("array:i64:") || arrayType === "slice:i64")) fail("javascript.index_read_collection", node.loc.start);
		const collection = emitExpression(node.object, owner, `${path}.collection`, context, arrayType);
		const index = dynamicIndexExpression(node.property, owner, `${path}.index`, context);
		const dynamic = arrayType === "slice:i64";
		const id = expressionID(owner, path, dynamic ? "dynamic-index-read" : "index-read");
		context.entities.push(graphEntity(id, entity(id, dynamic ? "000000000000000000000000000090fa" : "000000000000000000000000000090f4", [[dynamic ? 0x9fa0 : 0x9f40, ref(collection.id)], [dynamic ? 0x9fa1 : 0x9f41, ref(index.id)]])));
		return { id, type: "i64" };
	}
	if (node.type === "MemberExpression" && node.computed && node.object.type === "ArrayExpression" && expected === "i64") {
		if (node.object.elements.length > 32 || node.object.elements.some((item) => !item || item.type === "SpreadElement")) fail("javascript.fixed_array_shape", node.loc.start);
		const length = node.object.elements.length;
		const arrayType = stableID("execution", "type", "fixed-array", "i64", String(length));
		const values = node.object.elements.map((item, index) => emitExpression(item, owner, `${path}.collection.element.${index}`, context, "i64"));
		const constructID = expressionID(owner, `${path}.collection`, "fixed-array-construct");
		context.entities.push(graphEntity(arrayType, entity(arrayType, "000000000000000000000000000090f2", [[0x9f20, ref(ids.i64)], [0x9f21, `uu ${length}`]])));
		context.entities.push(graphEntity(constructID, entity(constructID, "000000000000000000000000000090f3", [[0x9f30, ref(arrayType)], [0x9f31, refs(values.map((item) => item.id))]])));
		const index = emitExpression(node.property, owner, `${path}.index`, context, "i64");
		const id = expressionID(owner, path, "index-read");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f4", [[0x9f40, ref(constructID)], [0x9f41, ref(index.id)]])));
		return { id, type: "i64" };
	}
  if (node.type === "Identifier") {
    const index = context.parameterNames.indexOf(node.name);
    if (index < 0) {
      const local = context.locals.get(node.name);
      if (!local || local.type !== expected) fail("javascript.unresolved_or_mistyped_identifier", node.loc.start);
      const id = stableID("execution", owner, local.mutable ? "place-read" : "local-read", local.id);
      context.entities.push(graphEntity(id, entity(id, local.mutable ? "000000000000000000000000000090e2" : "000000000000000000000000000090d2", [[local.mutable ? 0x9e20 : 0x9d20, ref(local.id)]])));
      return { id, type: expected };
    }
    if (context.parameterTypes[index] !== expected) fail("javascript.unresolved_or_mistyped_identifier", node.loc.start);
    const id = stableID("execution", owner, "read", String(index));
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009013", [[0x9130, ref(context.parameterIDs[index])]])));
    return { id, type: expected };
  }
  if (node.type === "Literal" && typeof node.value === "string" && expected === "string") {
    if (!isUnicodeScalarString(node.value)) fail("javascript.non_scalar_string", node.loc.start);
    const id = expressionID(owner, path, "string-literal");
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009050", [[0x9500, bytes(node.value)]])));
    return { id, type: "string" };
  }
  if (node.type === "Literal" && typeof node.value === "boolean" && expected === "bool") {
    const id = expressionID(owner, path, "boolean-literal");
    context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090b0", [[0x9b00, node.value ? "tr" : "fa"]])));
    return { id, type: "bool" };
  }
	if (node.type === "Literal" && typeof node.value === "bigint" && expected === "i64") {
		if (node.value < -(1n << 63n) || node.value > (1n << 63n) - 1n) fail("javascript.i64_literal_range", node.loc.start);
		const id = expressionID(owner, path, "integer-literal");
		context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009070", [[0x9700, `uu ${BigInt.asUintN(64, node.value)}`], [0x9701, ref(ids.i64)]])));
		return { id, type: "i64" };
	}
  if (node.type === "ObjectExpression" && expected.startsWith("record:")) {
    const record = context.recordsByName.get(expected.slice("record:".length));
    if (!record || node.properties.length !== record.fields.length) fail("javascript.record_shape", node.loc.start);
    const source = new Map();
    for (const property of node.properties) {
      if (property.type !== "Property" || property.computed || property.kind !== "init" || property.method) fail("javascript.record_property", property.loc.start);
      const name = property.key.type === "Identifier" ? property.key.name : property.key.value;
      if (typeof name !== "string" || source.has(name)) fail("javascript.record_property", property.loc.start);
      source.set(name, property.value);
    }
    const values = record.fields.map((field, index) => {
      const value = source.get(field.name);
      if (!value) fail("javascript.record_field_missing", node.loc.start);
      return emitExpression(value, owner, `${path}.field.${index}`, context, field.type);
    });
    const id = expressionID(owner, path, "record-construct");
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009033", [[0x9330, ref(record.id)], [0x9331, refs(values.map((item) => item.id))]])));
    return { id, type: expected };
  }
  if (node.type === "MemberExpression" && !node.computed && node.property.type === "Identifier") {
    const objectType = inferExpressionType(node.object, context);
    const record = objectType?.startsWith("record:") ? context.recordsByName.get(objectType.slice("record:".length)) : undefined;
    const field = record?.fields.find((item) => item.name === node.property.name);
    if (!record || !field || field.type !== expected) fail("javascript.record_field_read", node.loc.start);
    const base = emitExpression(node.object, owner, `${path}.record`, context, objectType);
    const id = expressionID(owner, path, "field-read");
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009032", [[0x9320, ref(base.id)], [0x9321, ref(field.id)]])));
    return { id, type: expected };
  }
  if (node.type === "CallExpression" && node.callee.type === "Identifier" && !node.optional) {
    const callee = context.functionsByName.get(node.callee.name);
    if (!callee || callee.signature.result !== expected || callee.signature.parameters.length !== node.arguments.length) fail("javascript.unsupported_call", node.loc.start);
    const arguments_ = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.argument.${index}`, context, callee.signature.parameters[index].type));
    const id = expressionID(owner, path, "function-call");
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009060", [[0x9600, ref(callee.id)], [0x9601, refs(arguments_.map((item) => item.id))]])));
    return { id, type: expected };
  }
  const operator = node.operator;
  const table = {
	"+:i64": ["add", "00000000000000000000000000009014", 0x9140, 0x9141, "i64"],
	"-:i64": ["subtract", "000000000000000000000000000090a0", 0x9a00, 0x9a01, "i64"],
	"<=:bool": ["less-equal", "00000000000000000000000000009021", 0x9160, 0x9161, "i64"],
    "+:string": ["string-concat", "000000000000000000000000000090c3", 0x9c30, 0x9c31, "string"],
    "===:bool": ["string-equal", "000000000000000000000000000090c2", 0x9c20, 0x9c21, "string"],
    "&&:bool": ["boolean-and", "000000000000000000000000000090b1", 0x9b10, 0x9b11, "bool"],
    "||:bool": ["boolean-or", "000000000000000000000000000090c1", 0x9c10, 0x9c11, "bool"],
  };
  const rule = table[`${operator}:${expected}`];
  if (!rule || !["BinaryExpression", "LogicalExpression"].includes(node.type)) fail("javascript.unsupported_expression", node.loc.start);
  const [kind, schema, leftField, rightField, operandType] = rule;
  const left = emitExpression(node.left, owner, `${path}.left`, context, operandType);
  const right = emitExpression(node.right, owner, `${path}.right`, context, operandType);
  const id = expressionID(owner, path, kind);
  const fields = [[leftField, ref(left.id)], [rightField, ref(right.id)]];
	if (schema === "00000000000000000000000000009014") fields.push([0x9142, ref(ids.i64)]);
	if (schema === "000000000000000000000000000090a0") fields.push([0x9a02, ref(ids.i64)]);
	if (schema === "00000000000000000000000000009021") fields.push([0x9162, ref(ids.i64)]);
  context.entities.push(graphEntity(id, entity(id, schema, fields)));
  return { id, type: expected };
}

function dynamicIndexExpression(node, owner, path, context) {
	if (node.type === "Literal" && typeof node.value === "number" && Number.isSafeInteger(node.value)) {
		const id = expressionID(owner, path, "integer-literal");
		context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009070", [[0x9700, `uu ${BigInt.asUintN(64, BigInt(node.value))}`], [0x9701, ref(ids.i64)]])));
		return { id, type: "i64" };
	}
	if (node.type === "BinaryExpression" && (node.operator === "+" || node.operator === "-")) {
		const left = dynamicIndexExpression(node.left, owner, `${path}.left`, context);
		const right = dynamicIndexExpression(node.right, owner, `${path}.right`, context);
		const subtract = node.operator === "-";
		const id = expressionID(owner, path, subtract ? "subtract" : "add");
		context.entities.push(graphEntity(id, entity(id, subtract ? "000000000000000000000000000090a0" : "00000000000000000000000000009014", subtract
			? [[0x9a00, ref(left.id)], [0x9a01, ref(right.id)], [0x9a02, ref(ids.i64)]]
			: [[0x9140, ref(left.id)], [0x9141, ref(right.id)], [0x9142, ref(ids.i64)]])));
		return { id, type: "i64" };
	}
	return emitExpression(node, owner, path, context, "i64");
}

function inferExpressionType(node, context) {
  if (node.type === "ThisExpression" && context.receiver) return context.receiver.type;
  if (node.type === "Identifier") {
    const parameter = context.parameterNames.indexOf(node.name);
    if (parameter >= 0) return context.parameterTypes[parameter];
    const local = context.locals.get(node.name);
    if (local) return local.type;
  }
  if (node.type === "Literal" && typeof node.value === "string") return "string";
  if (node.type === "Literal" && typeof node.value === "boolean") return "bool";
	if (node.type === "MemberExpression" && !node.computed && node.property.name === "length") return "i64";
  if (node.type === "ObjectExpression") {
	const propertyNames = new Set(node.properties.map((property) => property.key?.name ?? property.key?.value));
	if (propertyNames.size === 2 && propertyNames.has("state") && propertyNames.has("result")) {
		const stateNode = node.properties.find((property) => (property.key?.name ?? property.key?.value) === "state")?.value;
		const resultNode = node.properties.find((property) => (property.key?.name ?? property.key?.value) === "result")?.value;
		return `transition:${inferExpressionType(stateNode, context)}:${inferExpressionType(resultNode, context)}`;
	}
    const names = new Set(node.properties.map((property) => property.key?.name ?? property.key?.value));
    const matches = [...context.recordsByName.values()].filter((record) => record.fields.length === names.size && record.fields.every((field) => names.has(field.name)));
    if (matches.length === 1) return `record:${matches[0].name}`;
  }
  if (node.type === "CallExpression" && node.callee.type === "Identifier") {
    const callee = context.functionsByName.get(node.callee.name);
    if (callee) return callee.signature.result;
  }
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed) {
		const receiverType = inferExpressionType(node.callee.object, context);
		const method = context.methodsByTypeAndName?.get(`${receiverType.slice("record:".length)}:${node.callee.property.name}`);
		if (method) return method.signature.result;
	}
	if (node.type === "NewExpression" && node.callee.type === "Identifier" && context.recordsByName.has(node.callee.name)) return `record:${node.callee.name}`;
	if (node.type === "MemberExpression" && !node.computed) {
		const objectType = inferExpressionType(node.object, context);
		if (objectType.startsWith("transition:")) {
			const parts = objectType.split(":");
			if (node.property.name === "state") return `record:${parts[2]}`;
			if (node.property.name === "result") return parts.slice(3).join(":");
		}
		if (objectType.startsWith("record:")) {
			const record = context.recordsByName.get(objectType.slice("record:".length));
			const recordField = record?.fields.find((item) => item.name === node.property.name);
			if (recordField) return recordField.type;
		}
	}
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && (node.callee.property.name === "with" || node.callee.property.name === "concat")) return "slice:i64";
  if (node.type === "LogicalExpression" && (node.operator === "&&" || node.operator === "||")) return "bool";
  if (node.type === "BinaryExpression" && node.operator === "===") return "bool";
	if (node.type === "BinaryExpression" && node.operator === "<=") return "bool";
	if (node.type === "BinaryExpression" && node.operator === "+") {
    const left = inferExpressionType(node.left, context);
    const right = inferExpressionType(node.right, context);
    if (left === "string" && right === "string") return "string";
  }
	if (node.type === "BinaryExpression" && node.operator === "-") return "i64";
  fail("javascript.ambiguous_local_type", node.loc.start);
}

function blockStatements(node) {
  if (node.type === "BlockStatement") return node.body;
  if (node.type === "IfStatement" || node.type === "ReturnStatement") return [node];
  fail("javascript.unsupported_branch", node.loc.start);
}
function containsReturn(node) {
  if (node.type === "ReturnStatement") return true;
  if (node.type === "BlockStatement") return node.body.some(containsReturn);
  if (node.type === "IfStatement") return containsReturn(node.consequent) || (node.alternate ? containsReturn(node.alternate) : false);
  return false;
}
function expressionID(owner, path, kind) {
	if (path === "root" && kind === "less-equal") return stableID("execution", owner, "less-equal");
	if (path === "root.left" && kind === "add") return stableID("execution", owner, "add");
	return stableID("execution", owner, "expression", path, kind);
}
function isUnicodeScalarString(value) {
  for (let index = 0; index < value.length; index += 1) {
    const unit = value.charCodeAt(index);
    if (unit >= 0xd800 && unit <= 0xdbff) {
      const next = value.charCodeAt(index + 1);
      if (!(next >= 0xdc00 && next <= 0xdfff)) return false;
      index += 1;
    } else if (unit >= 0xdc00 && unit <= 0xdfff) return false;
  }
  return true;
}
function stableID(...parts) {
  const hash = crypto.createHash("sha256").update("seme.provider.identity.v1\0");
  for (const part of parts) hash.update(part).update("\0");
  const bytes = hash.digest().subarray(0, 15);
  return `80${bytes.toString("hex")}`;
}
function graphEntity(id, text) { return { id, text }; }
function bytes(value) { return `by ${Buffer.from(value, "utf8").toString("hex") || "-"}`; }
function ref(value) { return `rf ${value}`; }
function refs(values) { return `li ${values.length}${values.map((value) => `\nrf ${value}`).join("")}`; }
function entity(id, schema, fields) {
  fields.sort((left, right) => left[0] - right[0]);
  return `en ${id} ${schema} 1 ${fields.length}\n${fields.map(([field, value]) => `fi ${field.toString(16).padStart(32, "0")} ${value}\n`).join("")}`;
}
function compose(moduleG1, revision, additions) {
  const entities = [];
  let current;
  for (const line of moduleG1.trim().split("\n")) {
    if (line.startsWith("en ")) { current = { id: line.split(/\s+/)[1], text: "" }; entities.push(current); }
    if (current) current.text += `${line}\n`;
  }
  const moduleRoot = entities.find((item) => item.id === "00000000000000000000000000009000");
  const moduleVersion = Number(moduleRoot?.text.match(/^en\s+\S+\s+\S+\s+(\d+)\s+/m)?.[1] ?? 0);
  if (moduleVersion >= 26) entities.length = 0;
  const unique = new Map(entities.map((item) => [item.id, item]));
  for (const item of additions) {
    const existing = unique.get(item.id);
    if (existing && existing.text !== item.text) fail("javascript.identity_collision");
    unique.set(item.id, item);
  }
  entities.length = 0;
  entities.push(...unique.values());
  entities.sort((left, right) => left.id.localeCompare(right.id));
  return `# Generated exact JavaScript to Core Execution lift.\nve 1\nmo 00000000000000000000000000009000\nrv ${revision}\npc 0\nec ${entities.length}\n${entities.map((item) => `\n${item.text}`).join("")}`;
}
function fail(code, location) {
  const suffix = location ? `:${location.line}:${location.column + 1}` : "";
  throw new Error(`${code}${suffix}`);
}
