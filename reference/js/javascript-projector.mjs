const schema = {
  integerType: "00000000000000000000000000009010",
  function: "00000000000000000000000000009011",
  parameter: "00000000000000000000000000009012",
  read: "00000000000000000000000000009013",
  integerAdd: "00000000000000000000000000009014",
  program: "00000000000000000000000000009015",
  boolType: "00000000000000000000000000009020",
  stringType: "00000000000000000000000000009040",
  stringLiteral: "00000000000000000000000000009050",
  integerLiteral: "00000000000000000000000000009070",
  block: "00000000000000000000000000009080",
  returned: "00000000000000000000000000009081",
  boolLiteral: "000000000000000000000000000090b0",
  boolAnd: "000000000000000000000000000090b1",
  branch: "000000000000000000000000000090c0",
  boolOr: "000000000000000000000000000090c1",
  stringEqual: "000000000000000000000000000090c2",
  stringConcat: "000000000000000000000000000090c3",
  localBinding: "000000000000000000000000000090d0",
  bindLocal: "000000000000000000000000000090d1",
  localRead: "000000000000000000000000000090d2",
  call: "00000000000000000000000000009060",
  recordType: "00000000000000000000000000009030",
  recordField: "00000000000000000000000000009031",
  fieldRead: "00000000000000000000000000009032",
  recordConstruct: "00000000000000000000000000009033",
  mutablePlace: "000000000000000000000000000090e0",
  declarePlace: "000000000000000000000000000090e1",
  placeRead: "000000000000000000000000000090e2",
  assignPlace: "000000000000000000000000000090e3",
  whileLoop: "000000000000000000000000000090e4",
  when: "000000000000000000000000000090f0",
  effectInvoke: "000000000000000000000000000090f1",
  effect: "00000000000000000000000000000015",
  fixedArrayType: "000000000000000000000000000090f2",
  fixedArrayConstruct: "000000000000000000000000000090f3",
  indexRead: "000000000000000000000000000090f4",
  iterationBinding: "000000000000000000000000000090f5",
  iterationBindingRead: "000000000000000000000000000090f6",
  fold: "000000000000000000000000000090f7",
  sliceType: "000000000000000000000000000090f8",
};

export function projectJavaScript(canonicalG1) {
  const graph = parseG1(canonicalG1);
  const programs = [...graph.values()].filter((entity) => entity.schema === schema.program);
  if (programs.length !== 1) fail("javascript_projection.requires_one_program");
  const entryID = reference(field(programs[0], 0x9151));
  const functionIDs = references(field(programs[0], 0x9150));
  if (!functionIDs.includes(entryID) || functionIDs.length === 0) fail("javascript_projection.entry_membership");
  const functions = new Map(functionIDs.map((id) => [id, required(graph, id, schema.function)]));
  const records = new Map();
  for (const entity of graph.values()) {
    if (entity.schema !== schema.recordType) continue;
    const name = text(field(entity, 0x9300));
    if (!/^[A-Za-z_$][\w$]*$/.test(name)) fail("javascript_projection.invalid_record_name");
    const fields = references(field(entity, 0x9301)).map((fieldID, index) => {
      const recordField = required(graph, fieldID, schema.recordField);
      const position = unsigned(field(recordField, 0x9312));
      if (position !== BigInt(index)) fail("javascript_projection.record_field_order");
      return { id: fieldID, name: text(field(recordField, 0x9310)), type: typeName(reference(field(recordField, 0x9311)), graph) };
    });
    records.set(entity.id, { id: entity.id, name, fields });
  }
  const names = new Map();
  for (const [id, fn] of functions) {
    const name = text(field(fn, 0x9110));
    if (!/^[A-Za-z_$][\w$]*$/.test(name) || [...names.values()].includes(name)) fail("javascript_projection.invalid_function_name");
    names.set(id, name);
  }
  const typedefs = [...records.values()].map((record) => ["/**", ` * @typedef {Object} ${record.name}`, ...record.fields.map((item) => ` * @property {${item.type}} ${item.name}`), " */"].join("\n")).join("\n\n");
  const body = functionIDs.map((id) => projectFunction(id, functions.get(id), id === entryID, { graph, functions, names, records })).join("\n");
  return typedefs ? `${typedefs}\n\n${body}` : body;
}

function projectFunction(functionID, fn, exported, program) {
  const name = text(field(fn, 0x9110));
  const parameterIDs = references(field(fn, 0x9111));
  const parameters = parameterIDs.map((id) => {
    const parameter = required(program.graph, id, schema.parameter);
    const parameterName = text(field(parameter, 0x9120));
    if (!/^[A-Za-z_$][\w$]*$/.test(parameterName)) fail("javascript_projection.invalid_parameter_name");
    return { id, name: parameterName, type: typeName(reference(field(parameter, 0x9121)), program.graph) };
  });
  const result = typeName(reference(field(fn, 0x9112)), program.graph);
  const context = { ...program, parameters: new Map(parameters.map((item) => [item.id, item])), locals: new Map() };
  const body = projectBlock(reference(field(fn, 0x9113)), context, "  ");
  const jsdoc = ["/**", ...parameters.map((item) => ` * @param {${item.type}} ${item.name}`), ` * @returns {${result}}`, " */"];
  return `${jsdoc.join("\n")}\n${exported ? "export " : ""}function ${name}(${parameters.map((item) => item.name).join(", ")}) {\n${body}\n}\n`;
}

function projectBlock(id, context, indent) {
  const block = required(context.graph, id, schema.block);
  const statements = references(field(block, 0x9800));
  if (statements.length === 0) fail("javascript_projection.block_arity");
  const localContext = { ...context, locals: new Map(context.locals) };
  const lines = [];
  for (let index = 0; index < statements.length; index += 1) {
    const statement = required(context.graph, statements[index]);
    if (statement.schema === schema.bindLocal || statement.schema === schema.declarePlace) {
	  const mutable = statement.schema === schema.declarePlace;
	  const binding = required(context.graph, reference(field(statement, mutable ? 0x9e10 : 0x9d10)), mutable ? schema.mutablePlace : schema.localBinding);
	  const nameField = mutable ? 0x9e00 : 0x9d00;
	  const typeField = mutable ? 0x9e01 : 0x9d01;
	  const initializerField = mutable ? 0x9e02 : 0x9d02;
	  const name = text(field(binding, nameField));
	  if (!/^[A-Za-z_$][\w$]*$/.test(name) || [...localContext.locals.values()].some((item) => item.name === name)) fail("javascript_projection.invalid_local_name");
	  const valueType = typeName(reference(field(binding, typeField)), context.graph);
	  const initializer = projectExpression(reference(field(binding, initializerField)), localContext);
	  lines.push(`${indent}${mutable ? "let" : "const"} ${name} = ${initializer};`);
	  localContext.locals.set(binding.id, { name, type: valueType, mutable });
	  continue;
    }
    if (statement.schema === schema.assignPlace) {
      const place = localContext.locals.get(reference(field(statement, 0x9e30)));
      if (!place?.mutable) fail("javascript_projection.assignment_target");
      lines.push(`${indent}${place.name} = ${projectExpression(reference(field(statement, 0x9e31)), localContext)};`);
      continue;
    }
    if (statement.schema === schema.whileLoop) {
      const condition = projectExpression(reference(field(statement, 0x9e40)), localContext);
      const body = projectBlock(reference(field(statement, 0x9e41)), localContext, `${indent}  `);
      lines.push(`${indent}while (${condition}) {\n${body}\n${indent}}`);
      continue;
    }
    if (statement.schema === schema.when) {
      const condition = projectExpression(reference(field(statement, 0x9f00)), localContext);
      const body = projectBlock(reference(field(statement, 0x9f01)), localContext, `${indent}  `);
      lines.push(`${indent}if (${condition}) {\n${body}\n${indent}}`);
      continue;
    }
    if (statement.schema === schema.effectInvoke) {
      const effect = required(context.graph, reference(field(statement, 0x9f10)), schema.effect);
      if (text(field(effect, 0x150)) !== "observability.log") fail("javascript_projection.unsupported_effect");
      const arguments_ = references(field(statement, 0x9f11));
      if (arguments_.length !== 1) fail("javascript_projection.effect_arity");
      lines.push(`${indent}console.log(${projectExpression(arguments_[0], localContext)});`);
      continue;
    }
    if (statement.schema === schema.returned) {
      const values = references(field(statement, 0x9810));
      if (values.length !== 1 || index !== statements.length - 1) fail("javascript_projection.return_arity");
      lines.push(`${indent}return ${projectExpression(values[0], localContext)};`);
      continue;
    }
    if (statement.schema === schema.branch) {
      if (index !== statements.length - 1) fail("javascript_projection.branch_terminal");
      const condition = projectExpression(reference(field(statement, 0x9c00)), localContext);
      const thenBody = projectBlock(reference(field(statement, 0x9c01)), localContext, `${indent}  `);
      const elseBody = projectBlock(reference(field(statement, 0x9c02)), localContext, `${indent}  `);
      lines.push(`${indent}if (${condition}) {\n${thenBody}\n${indent}} else {\n${elseBody}\n${indent}}`);
      continue;
    }
    fail("javascript_projection.unsupported_statement");
  }
  return lines.join("\n");
}

function projectExpression(id, context) {
  const expression = required(context.graph, id);
  if (expression.schema === schema.read) {
    const parameter = context.parameters.get(reference(field(expression, 0x9130)));
    if (!parameter) fail("javascript_projection.unknown_parameter");
    return parameter.name;
  }
  if (expression.schema === schema.localRead) {
    const local = context.locals.get(reference(field(expression, 0x9d20)));
    if (!local) fail("javascript_projection.local_out_of_scope");
    return local.name;
  }
  if (expression.schema === schema.placeRead) {
    const local = context.locals.get(reference(field(expression, 0x9e20)));
    if (!local?.mutable) fail("javascript_projection.place_out_of_scope");
    return local.name;
  }
	if (expression.schema === schema.iterationBindingRead) {
		const binding = context.iterationBindings?.get(reference(field(expression, 0x9f60)));
		if (!binding) fail("javascript_projection.iteration_binding_scope");
		return binding;
	}
	if (expression.schema === schema.integerLiteral) {
		const value = BigInt.asIntN(64, unsigned(field(expression, 0x9700)));
		return `${value}n`;
	}
	if (expression.schema === schema.fold) {
		const accumulatorID = reference(field(expression, 0x9f72));
		const elementID = reference(field(expression, 0x9f73));
		const accumulator = required(context.graph, accumulatorID, schema.iterationBinding);
		const element = required(context.graph, elementID, schema.iterationBinding);
		const accumulatorName = text(field(accumulator, 0x9f50));
		const elementName = text(field(element, 0x9f50));
		if (!/^[A-Za-z_$][\w$]*$/.test(accumulatorName) || !/^[A-Za-z_$][\w$]*$/.test(elementName) || accumulatorName === elementName) fail("javascript_projection.iteration_binding_name");
		const foldContext = { ...context, iterationBindings: new Map(context.iterationBindings || []) };
		foldContext.iterationBindings.set(accumulatorID, accumulatorName);
		foldContext.iterationBindings.set(elementID, elementName);
		return `${projectExpression(reference(field(expression, 0x9f70)), context)}.reduce((${accumulatorName}, ${elementName}) => ${projectExpression(reference(field(expression, 0x9f74)), foldContext)}, ${projectExpression(reference(field(expression, 0x9f71)), context)})`;
	}
  if (expression.schema === schema.stringLiteral) return JSON.stringify(text(field(expression, 0x9500)));
  if (expression.schema === schema.boolLiteral) return atom(field(expression, 0x9b00)) === "tr" ? "true" : "false";
  if (expression.schema === schema.call) {
    const callee = reference(field(expression, 0x9600));
    if (!context.functions.has(callee)) fail("javascript_projection.call_outside_program");
    const arguments_ = references(field(expression, 0x9601)).map((argument) => projectExpression(argument, context));
    return `${context.names.get(callee)}(${arguments_.join(", ")})`;
  }
  if (expression.schema === schema.recordConstruct) {
    const record = context.records.get(reference(field(expression, 0x9330)));
    const values = references(field(expression, 0x9331));
    if (!record || values.length !== record.fields.length) fail("javascript_projection.record_construct");
    return `{ ${record.fields.map((item, index) => `${item.name}: ${projectExpression(values[index], context)}`).join(", ")} }`;
  }
  if (expression.schema === schema.fieldRead) {
    const fieldID = reference(field(expression, 0x9321));
    const record = [...context.records.values()].find((item) => item.fields.some((candidate) => candidate.id === fieldID));
    const recordField = record?.fields.find((item) => item.id === fieldID);
    if (!recordField) fail("javascript_projection.record_field");
    return `${projectExpression(reference(field(expression, 0x9320)), context)}.${recordField.name}`;
  }
  if (expression.schema === schema.indexRead) {
    const collectionID = reference(field(expression, 0x9f40));
    const collection = required(context.graph, collectionID);
    let rendered;
    if (collection.schema === schema.fixedArrayConstruct) {
      const type = required(context.graph, reference(field(collection, 0x9f30)), schema.fixedArrayType);
      if (reference(field(type, 0x9f20)) !== [...context.graph.values()].find((item) => item.schema === schema.integerType)?.id) fail("javascript_projection.fixed_array_element_type");
      const values = references(field(collection, 0x9f31));
      if (BigInt(values.length) !== unsigned(field(type, 0x9f21))) fail("javascript_projection.fixed_array_length");
      rendered = `[${values.map((value) => projectExpression(value, context)).join(", ")}]`;
    } else {
      rendered = projectExpression(collectionID, context);
    }
    return `${rendered}[${projectExpression(reference(field(expression, 0x9f41)), context)}]`;
  }
  const binary = new Map([
	[schema.integerAdd, [0x9140, 0x9141, "+"]],
    [schema.boolAnd, [0x9b10, 0x9b11, "&&"]],
    [schema.boolOr, [0x9c10, 0x9c11, "||"]],
    [schema.stringEqual, [0x9c20, 0x9c21, "==="]],
    [schema.stringConcat, [0x9c30, 0x9c31, "+"]],
  ]).get(expression.schema);
  if (!binary) fail("javascript_projection.unsupported_expression");
  const [leftField, rightField, operator] = binary;
  return `(${projectExpression(reference(field(expression, leftField)), context)} ${operator} ${projectExpression(reference(field(expression, rightField)), context)})`;
}

function typeName(id, graph) {
  const type = required(graph, id);
  if (type.schema === schema.integerType) return "bigint";
  if (type.schema === schema.stringType) return "string";
  if (type.schema === schema.boolType) return "boolean";
  if (type.schema === schema.recordType) return text(field(type, 0x9300));
	if (type.schema === schema.fixedArrayType) {
		const element = required(graph, reference(field(type, 0x9f20)));
		if (element.schema !== schema.integerType) fail("javascript_projection.fixed_array_element_type");
		const length = unsigned(field(type, 0x9f21));
		if (length < 0n || length > 32n) fail("javascript_projection.fixed_array_length");
		return `bigint[${length}]`;
	}
	if (type.schema === schema.sliceType) {
		const element = required(graph, reference(field(type, 0x9f80)));
		if (element.schema !== schema.integerType) fail("javascript_projection.slice_element_type");
		return "bigint[]";
	}
  fail("javascript_projection.unsupported_type");
}

function parseG1(source) {
  const lines = source.trim().split("\n");
  const graph = new Map();
  for (let index = 0; index < lines.length;) {
    if (!lines[index].startsWith("en ")) { index += 1; continue; }
    const header = lines[index].trim().split(/\s+/);
    const entity = { id: header[1], schema: header[2], fields: new Map() };
    index += 1;
    for (let count = 0; count < Number(header[4]); count += 1) {
      const match = /^fi ([0-9a-f]{32}) (.*)$/.exec(lines[index]);
      if (!match) fail("javascript_projection.malformed_field");
      const value = [match[2]];
      const list = /^li (\d+)$/.exec(match[2]);
      index += 1;
      if (list) {
        for (let item = 0; item < Number(list[1]); item += 1, index += 1) value.push(lines[index].trim());
      }
      const fieldID = BigInt(`0x${match[1]}`);
      if (entity.fields.has(fieldID)) fail("javascript_projection.duplicate_field");
      entity.fields.set(fieldID, value);
    }
    if (graph.has(entity.id)) fail("javascript_projection.duplicate_entity");
    graph.set(entity.id, entity);
  }
  return graph;
}
function required(graph, id, expected) {
  const entity = graph.get(id);
  if (!entity || (expected && entity.schema !== expected)) fail("javascript_projection.missing_or_mistyped_entity");
  return entity;
}
function field(entity, id) {
  const value = entity.fields.get(BigInt(id));
  if (!value) fail("javascript_projection.missing_field");
  return value;
}
function atom(value) { return value[0]; }
function unsigned(value) {
  const match = /^uu (\d+)$/.exec(atom(value));
  if (!match) fail("javascript_projection.invalid_unsigned");
  return BigInt(match[1]);
}
function reference(value) {
  const match = /^rf ([0-9a-f]{32})$/.exec(atom(value));
  if (!match) fail("javascript_projection.invalid_reference");
  return match[1];
}
function references(value) {
  const match = /^li (\d+)$/.exec(atom(value));
  if (!match || value.length !== Number(match[1]) + 1) fail("javascript_projection.invalid_reference_list");
  return value.slice(1).map((item) => reference([item]));
}
function text(value) {
  const match = /^by ([-0-9a-f]*)$/.exec(atom(value));
  if (!match) fail("javascript_projection.invalid_bytes");
  if (match[1] === "-") return "";
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(Buffer.from(match[1], "hex"));
  } catch {
    fail("javascript_projection.invalid_utf8");
  }
}
function fail(code) { throw new Error(code); }
