const schema = {
  integerType: "00000000000000000000000000009010",
  function: "00000000000000000000000000009011",
  parameter: "00000000000000000000000000009012",
  read: "00000000000000000000000000009013",
  integerAdd: "00000000000000000000000000009014",
  integerMultiply: "00000000000000000000000000009090",
  integerLessEqual: "00000000000000000000000000009021",
  program: "00000000000000000000000000009015",
  boolType: "00000000000000000000000000009020",
  stringType: "00000000000000000000000000009040",
  bytesType: "00000000000000000000000000009041",
  resultType: "00000000000000000000000000009042",
  resultOk: "00000000000000000000000000009043",
  resultError: "00000000000000000000000000009044",
  stringLiteral: "00000000000000000000000000009050",
  integerLiteral: "00000000000000000000000000009070",
  block: "00000000000000000000000000009080",
  returned: "00000000000000000000000000009081",
  boolLiteral: "000000000000000000000000000090b0",
  integerSubtract: "000000000000000000000000000090a0",
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
  nativeDefer: "0000000000000000000000000000a075",
  nativeFieldAssignment: "0000000000000000000000000000a076",
  nativeSwitchCase: "0000000000000000000000000000a077",
  nativeSwitch: "0000000000000000000000000000a078",
  unitValue: "0000000000000000000000000000a06b",
  nativeBranch: "0000000000000000000000000000a079",
  nativeAddress: "0000000000000000000000000000a07a",
  nativeRangeBinding: "0000000000000000000000000000a07b",
  nativeRange: "0000000000000000000000000000a07c",
  nativeSlice: "0000000000000000000000000000a07d",
  nativeDereference: "0000000000000000000000000000a07e",
  nativeBinary: "0000000000000000000000000000a07f",
  nativeInvocation: "0000000000000000000000000000a06d",
  nativeType: "0000000000000000000000000000a071",
  when: "000000000000000000000000000090f0",
  effectInvoke: "000000000000000000000000000090f1",
  effect: "00000000000000000000000000000015",
  capability: "00000000000000000000000000000016",
  fixedArrayType: "000000000000000000000000000090f2",
  fixedArrayConstruct: "000000000000000000000000000090f3",
  indexRead: "000000000000000000000000000090f4",
  iterationBinding: "000000000000000000000000000090f5",
  iterationBindingRead: "000000000000000000000000000090f6",
  fold: "000000000000000000000000000090f7",
  sliceType: "000000000000000000000000000090f8",
  collectionLength: "000000000000000000000000000090f9",
  dynamicIndexRead: "000000000000000000000000000090fa",
  collectionAppend: "000000000000000000000000000090fb",
  collectionUpdate: "000000000000000000000000000090fc",
  receiverBinding: "0000000000000000000000000000a000",
  receiverRead: "0000000000000000000000000000a001",
  method: "0000000000000000000000000000a002",
  methodCall: "0000000000000000000000000000a003",
  transitionType: "0000000000000000000000000000a004",
  stateTransition: "0000000000000000000000000000a005",
  transitionState: "0000000000000000000000000000a006",
  transitionResult: "0000000000000000000000000000a007",
  interfaceType: "0000000000000000000000000000a010",
  methodRequirement: "0000000000000000000000000000a011",
  satisfactionWitness: "0000000000000000000000000000a012",
  interfaceValue: "0000000000000000000000000000a013",
  dynamicMethodCall: "0000000000000000000000000000a014",
  functionType: "0000000000000000000000000000a020",
  captureBinding: "0000000000000000000000000000a021",
  captureRead: "0000000000000000000000000000a022",
  closureConstruct: "0000000000000000000000000000a023",
  indirectCall: "0000000000000000000000000000a024",
  mutableCaptureBinding: "0000000000000000000000000000a030",
  mutableCaptureRead: "0000000000000000000000000000a031",
  captureUpdate: "0000000000000000000000000000a032",
  sequence: "0000000000000000000000000000a033",
  mutableClosureConstruct: "0000000000000000000000000000a034",
  statefulIndirectCall: "0000000000000000000000000000a035",
  mapType: "0000000000000000000000000000a040",
  emptyMap: "0000000000000000000000000000a041",
  mapLookup: "0000000000000000000000000000a042",
  mapLookupOption: "0000000000000000000000000000a044",
  mapUpdate: "0000000000000000000000000000a043",
  mapRemove: "0000000000000000000000000000a067",
  sliceRemove: "0000000000000000000000000000a066",
  sliceConstruct: "0000000000000000000000000000a068",
  optionType: "0000000000000000000000000000a050",
  optionNone: "0000000000000000000000000000a051",
  optionSome: "0000000000000000000000000000a052",
  bytesLiteral: "0000000000000000000000000000a064",
  bytesEqual: "0000000000000000000000000000a065",
  variantBinding: "0000000000000000000000000000a060",
  variantBindingRead: "0000000000000000000000000000a061",
  resultMatch: "0000000000000000000000000000a062",
  optionMatch: "0000000000000000000000000000a063",
};

export function projectJavaScript(canonicalG1) {
  const graph = parseG1(canonicalG1);
  const programs = [...graph.values()].filter((entity) => entity.schema === schema.program);
  if (programs.length !== 1) fail("javascript_projection.requires_one_program");
  const entryID = reference(field(programs[0], 0x9151));
  const functionIDs = references(field(programs[0], 0x9150));
  if (!functionIDs.includes(entryID) || functionIDs.length === 0) fail("javascript_projection.entry_membership");
  const functions = new Map(functionIDs.map((id) => [id, required(graph, id, schema.function)]));
  const methods = new Map([...graph.values()].filter((entity) => entity.schema === schema.method).map((entity) => [entity.id, entity]));
  const interfaces = new Map();
  for (const entity of graph.values()) {
    if (entity.schema !== schema.interfaceType) continue;
    const name = text(field(entity, 0xa0100));
    if (!/^[A-Za-z_$][\w$]*$/.test(name)) fail("javascript_projection.invalid_interface_name");
    const requirements = references(field(entity, 0xa0101)).map((id) => {
      const requirement = required(graph, id, schema.methodRequirement);
      return { id, name: text(field(requirement, 0xa0110)), parameters: references(field(requirement, 0xa0111)).map((type) => typeName(type, graph)), result: typeName(reference(field(requirement, 0xa0112)), graph) };
    });
    interfaces.set(entity.id, { id: entity.id, name, requirements });
  }
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
  const program = { graph, functions, names, records, methods, interfaces };
  const interfaceDocs = [...interfaces.values()].map((interface_) => ["/**", ` * @interface ${interface_.name}`, ...interface_.requirements.flatMap((requirement) => [` * @method ${requirement.name}`, ...requirement.parameters.map((type, index) => ` * @param {${type}} argument${index}`), ` * @returns {${requirement.result}}`]), " */"].join("\n")).join("\n\n");
  const typedefs = [...records.values()].map((record) => ["/**", ` * @typedef {Object} ${record.name}`, ...record.fields.map((item) => ` * @property {${item.type}} ${item.name}`), " */"].join("\n")).join("\n\n");
  const classes = [...records.values()].map((record) => projectClass(record, program)).filter(Boolean).join("\n");
  const body = functionIDs.map((id) => projectFunction(id, functions.get(id), id === entryID, program)).join("\n");
  return [interfaceDocs, typedefs, classes, body].filter(Boolean).join("\n\n");
}

function projectClass(record, program) {
  const methods = [...program.methods.entries()].filter(([, method]) => {
    const receiver = required(program.graph, reference(field(method, 0xa0021)), schema.receiverBinding);
    return reference(field(receiver, 0xa0001)) === record.id;
  });
  if (!methods.length) return "";
  const constructor = `  constructor(${record.fields.map((item) => item.name).join(", ")}) {\n${record.fields.map((item) => `    this.${item.name} = ${item.name};`).join("\n")}\n  }`;
  return `class ${record.name} {\n${constructor}\n\n${methods.map(([id, method]) => projectMethod(id, method, program)).join("\n\n")}\n}`;
}

function projectMethod(methodID, method, program) {
  const name = text(field(method, 0xa0020));
  if (!/^[A-Za-z_$][\w$]*$/.test(name)) fail("javascript_projection.invalid_method_name");
  const receiver = required(program.graph, reference(field(method, 0xa0021)), schema.receiverBinding);
  const parameterIDs = references(field(method, 0xa0022));
  const parameters = parameterIDs.map((id) => {
    const parameter = required(program.graph, id, schema.parameter);
    return { id, name: text(field(parameter, 0x9120)), type: typeName(reference(field(parameter, 0x9121)), program.graph) };
  });
  const result = typeName(reference(field(method, 0xa0023)), program.graph);
  const context = { ...program, receiver: { id: receiver.id, name: "this" }, parameters: new Map(parameters.map((item) => [item.id, item])), locals: new Map() };
  const body = projectBlock(reference(field(method, 0xa0024)), context, "    ");
  const jsdoc = ["  /**", ...parameters.map((item) => `   * @param {${item.type}} ${item.name}`), `   * @returns {${result}}`, "   */"].join("\n");
  return `${jsdoc}\n  ${name}(${parameters.map((item) => item.name).join(", ")}) {\n${body}\n  }`;
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
  const mutableInvocation = projectMutableInvocationBlock(statements, context, indent);
  if (mutableInvocation) return mutableInvocation;
  const localContext = { ...context, locals: new Map(context.locals) };
  const lines = [];
  for (let index = 0; index < statements.length; index += 1) {
    const statefulAssignment = projectStatefulAssignment(statements, index, localContext);
    if (statefulAssignment) {
      lines.push(`${indent}${statefulAssignment.line}`);
      index += statefulAssignment.consumed - 1;
      continue;
    }
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
    if (statement.schema === schema.nativeRange) {
      const language = text(field(statement, 0xa07c0));
      const collection = projectExpression(reference(field(statement, 0xa07c1)), localContext);
      const child = { ...localContext, locals: new Map(localContext.locals) };
      const binding = (fieldID) => { const ids = references(field(statement, fieldID)); if (ids.length > 1) fail("javascript_projection.native_range_binding"); if (!ids.length) return null; const item = required(context.graph, ids[0], schema.nativeRangeBinding); const name = text(field(item, 0xa07b0)); child.locals.set(ids[0], { name, type: typeName(reference(field(item, 0xa07b1)), context.graph), mutable: false }); return name; };
      const key = binding(0xa07c2), value = binding(0xa07c3);
      const body = projectBlock(reference(field(statement, 0xa07c4)), child, `${indent}    `);
      lines.push(`${indent}Seme.nativeRange(${JSON.stringify(language)}, ${collection}, ${JSON.stringify(key)}, ${JSON.stringify(value)}, (${key ?? "_key"}, ${value ?? "_value"}) => {\n${body}\n${indent}});`);
      continue;
    }
    if (statement.schema === schema.nativeDefer) {
      const language = text(field(statement, 0xa0750));
      const invocation = projectExpression(reference(field(statement, 0xa0751)), localContext);
      lines.push(`${indent}Seme.deferNative(${JSON.stringify(language)}, () => ${invocation});`);
      continue;
    }
    if (statement.schema === schema.nativeFieldAssignment) {
      const language = text(field(statement, 0xa0760));
      const member = text(field(statement, 0xa0761));
      const receiver = projectExpression(reference(field(statement, 0xa0762)), localContext);
      const value = projectExpression(reference(field(statement, 0xa0763)), localContext);
      lines.push(`${indent}Seme.assignNativeField(${JSON.stringify(language)}, ${receiver}, ${JSON.stringify(member)}, ${value});`);
      continue;
    }
    if (statement.schema === schema.nativeSwitch) {
      const language = text(field(statement, 0xa0780));
      const subjectID = reference(field(statement, 0xa0781));
      const subject = required(context.graph, subjectID).schema === schema.unitValue ? "undefined" : projectExpression(subjectID, localContext);
      const cases = references(field(statement, 0xa0782)).map((caseID) => {
        const branch = required(context.graph, caseID, schema.nativeSwitchCase);
        const values = references(field(branch, 0xa0770)).map((valueID) => projectExpression(valueID, localContext));
        const body = projectBlock(reference(field(branch, 0xa0771)), localContext, `${indent}    `);
        return `${indent}  { values: [${values.join(", ")}], run: () => {\n${body}\n${indent}  } }`;
      });
      const defaults = references(field(statement, 0xa0783));
      if (defaults.length > 1) fail("javascript_projection.native_switch_default");
      const fallback = defaults.length === 1 ? projectBlock(defaults[0], localContext, `${indent}    `) : "";
      lines.push(`${indent}Seme.nativeSwitch(${JSON.stringify(language)}, ${subject}, [\n${cases.join(",\n")}\n${indent}], () => {\n${fallback}\n${indent}});`);
      continue;
    }
    if (statement.schema === schema.nativeBranch) {
      const language = text(field(statement, 0xa0790));
      const operation = text(field(statement, 0xa0791));
      const target = text(field(statement, 0xa0792));
      lines.push(`${indent}Seme.nativeBranch(${JSON.stringify(language)}, ${JSON.stringify(operation)}, ${JSON.stringify(target)});`);
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
      const capability = required(context.graph, reference(field(effect, 0x151)), schema.capability);
      if (text(field(capability, 0x160)) !== "observability.log") fail("javascript_projection.effect_authority");
      const arguments_ = references(field(statement, 0x9f11));
      if (arguments_.length !== 1) fail("javascript_projection.effect_arity");
      lines.push(`${indent}console.log(${projectExpression(arguments_[0], localContext)});`);
      continue;
    }
    if (statement.schema === schema.returned) {
      const values = references(field(statement, 0x9810));
      if (values.length !== 1 || index !== statements.length - 1) fail("javascript_projection.return_arity");
      const returned = required(context.graph, values[0]);
      if (returned.schema === schema.mutableClosureConstruct) {
        const mutable = projectMutableClosure(returned, localContext, indent);
        lines.push(`${indent}let ${mutable.captureName} = ${mutable.initial};`);
        lines.push(`${indent}return ${mutable.closure};`);
        continue;
      }
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

function projectStatefulAssignment(statementIDs, index, context) {
  if (index + 2 >= statementIDs.length) return undefined;
  const bind = required(context.graph, statementIDs[index]);
  const updateClosure = required(context.graph, statementIDs[index + 1]);
  const updateTarget = required(context.graph, statementIDs[index + 2]);
  if (bind.schema !== schema.bindLocal || updateClosure.schema !== schema.assignPlace || updateTarget.schema !== schema.assignPlace) return undefined;
  const temporary = required(context.graph, reference(field(bind, 0x9d10)), schema.localBinding);
  const call = required(context.graph, reference(field(temporary, 0x9d02)), schema.statefulIndirectCall);
  const closure = context.locals.get(reference(field(updateClosure, 0x9e30)));
  const target = context.locals.get(reference(field(updateTarget, 0x9e30)));
  if (!closure?.mutable || !target?.mutable) fail("javascript_projection.stateful_assignment_target");
  const state = required(context.graph, reference(field(updateClosure, 0x9e31)), schema.transitionState);
  const result = required(context.graph, reference(field(updateTarget, 0x9e31)), schema.transitionResult);
  const stateRead = required(context.graph, reference(field(state, 0xa0060)), schema.localRead);
  const resultRead = required(context.graph, reference(field(result, 0xa0070)), schema.localRead);
  if (reference(field(stateRead, 0x9d20)) !== temporary.id || reference(field(resultRead, 0x9d20)) !== temporary.id) fail("javascript_projection.stateful_assignment_transition");
  const callee = required(context.graph, reference(field(call, 0xa0350)), schema.placeRead);
  if (reference(field(callee, 0x9e20)) !== reference(field(updateClosure, 0x9e30))) fail("javascript_projection.stateful_assignment_closure");
  const arguments_ = references(field(call, 0xa0351)).map((id) => projectExpression(id, context));
  return { consumed: 3, line: `${target.name} = ${closure.name}(${arguments_.join(", ")});` };
}

function projectMutableInvocationBlock(statementIDs, context, indent) {
  if (statementIDs.length !== 4) return undefined;
  const [declare, bind, assign, returned] = statementIDs.map((id) => required(context.graph, id));
  if (declare.schema !== schema.declarePlace || bind.schema !== schema.bindLocal || assign.schema !== schema.assignPlace || returned.schema !== schema.returned) return undefined;
  const counter = required(context.graph, reference(field(declare, 0x9e10)), schema.mutablePlace);
  if (reference(field(assign, 0x9e30)) !== counter.id) return undefined;
  const first = required(context.graph, reference(field(bind, 0x9d10)), schema.localBinding);
  const firstCall = required(context.graph, reference(field(first, 0x9d02)), schema.statefulIndirectCall);
  const assignedState = required(context.graph, reference(field(assign, 0x9e31)), schema.transitionState);
  const firstRead = required(context.graph, reference(field(assignedState, 0xa0060)), schema.localRead);
  if (reference(field(firstRead, 0x9d20)) !== first.id) return undefined;
  const values = references(field(returned, 0x9810));
  if (values.length !== 1) return undefined;
  const result = required(context.graph, values[0], schema.transitionResult);
  const secondCall = required(context.graph, reference(field(result, 0xa0070)), schema.statefulIndirectCall);
  const counterName = text(field(counter, 0x9e00));
  const initial = projectExpression(reference(field(counter, 0x9e02)), context);
  const firstArguments = references(field(firstCall, 0xa0351)).map((id) => projectExpression(id, context));
  const secondArguments = references(field(secondCall, 0xa0351)).map((id) => projectExpression(id, context));
  return [`${indent}const ${counterName} = ${initial};`, `${indent}${counterName}(${firstArguments.join(", ")});`, `${indent}return ${counterName}(${secondArguments.join(", ")});`].join("\n");
}

function projectMutableClosure(expression, context) {
  const captures = references(field(expression, 0xa0342));
  const parameterIDs = references(field(expression, 0xa0341));
  if (captures.length !== 1 || parameterIDs.length !== 1) fail("javascript_projection.mutable_closure_arity");
  const capture = required(context.graph, captures[0], schema.mutableCaptureBinding);
  const parameter = required(context.graph, parameterIDs[0], schema.parameter);
  const sequence = required(context.graph, reference(field(expression, 0xa0343)), schema.sequence);
  const steps = references(field(sequence, 0xa0330));
  if (steps.length !== 1) fail("javascript_projection.mutable_closure_sequence");
  const update = required(context.graph, steps[0], schema.captureUpdate);
  if (reference(field(update, 0xa0320)) !== capture.id) fail("javascript_projection.mutable_capture_update");
  const result = required(context.graph, reference(field(sequence, 0xa0331)), schema.mutableCaptureRead);
  if (reference(field(result, 0xa0310)) !== capture.id) fail("javascript_projection.mutable_capture_result");
  const captureName = text(field(capture, 0xa0300));
  const parameterName = text(field(parameter, 0x9120));
  const closureContext = { ...context, captures: new Map([[capture.id, { id: capture.id, name: captureName, mutable: true }]]), parameters: new Map([[parameter.id, { id: parameter.id, name: parameterName }]]), locals: new Map() };
  return { captureName, initial: projectExpression(reference(field(capture, 0xa0302)), context), closure: `(${parameterName}) => { ${captureName} = ${projectExpression(reference(field(update, 0xa0321)), closureContext)}; return ${captureName}; }` };
}

function projectExpression(id, context) {
  const expression = required(context.graph, id);
  if (expression.schema === schema.nativeInvocation) return `Seme.nativeInvoke(${JSON.stringify(text(field(expression, 0xa06d0)))}, ${JSON.stringify(text(field(expression, 0xa06d1)))}, ${JSON.stringify(text(field(expression, 0xa06d2)))}, [${references(field(expression, 0xa06d3)).map(value => projectExpression(value, context)).join(", ")}], ${JSON.stringify(reference(field(expression, 0xa06d4)))})`;
  if (expression.schema === schema.nativeAddress) return `Seme.nativeAddress(${JSON.stringify(text(field(expression, 0xa07a0)))}, ${projectExpression(reference(field(expression, 0xa07a1)), context)}, ${JSON.stringify(reference(field(expression, 0xa07a2)))})`;
  if (expression.schema === schema.nativeSlice) { const optional=(id)=>{const values=references(field(expression,id));if(values.length>1)fail("javascript_projection.native_slice_bound");return values.length?projectExpression(values[0],context):"undefined";}; return `Seme.nativeSlice(${JSON.stringify(text(field(expression,0xa07d0)))}, ${projectExpression(reference(field(expression,0xa07d1)),context)}, ${optional(0xa07d2)}, ${optional(0xa07d3)}, ${optional(0xa07d4)}, ${JSON.stringify(reference(field(expression,0xa07d5)))})`; }
  if (expression.schema === schema.nativeDereference) return `Seme.nativeDereference(${JSON.stringify(text(field(expression, 0xa07e0)))}, ${projectExpression(reference(field(expression, 0xa07e1)), context)}, ${JSON.stringify(reference(field(expression, 0xa07e2)))})`;
  if (expression.schema === schema.nativeBinary) return `Seme.nativeBinary(${JSON.stringify(text(field(expression, 0xa07f0)))}, ${JSON.stringify(text(field(expression, 0xa07f1)))}, ${projectExpression(reference(field(expression, 0xa07f2)), context)}, ${projectExpression(reference(field(expression, 0xa07f3)), context)}, ${JSON.stringify(reference(field(expression, 0xa07f4)))})`;
  if (expression.schema === schema.captureRead) {
    const capture = context.captures?.get(reference(field(expression, 0xa0220)));
    if (!capture) fail("javascript_projection.capture_scope");
    return capture.name;
  }
  if (expression.schema === schema.mutableCaptureRead) {
    const capture = context.captures?.get(reference(field(expression, 0xa0310)));
    if (!capture?.mutable) fail("javascript_projection.mutable_capture_scope");
    return capture.name;
  }
  if (expression.schema === schema.receiverRead) {
    if (!context.receiver || reference(field(expression, 0xa0010)) !== context.receiver.id) fail("javascript_projection.receiver_scope");
    return context.receiver.name;
  }
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
	if (expression.schema === schema.bytesLiteral) return `Seme.bytes([${byteValues(field(expression, 0xa0640)).join(", ")}])`;
	if (expression.schema === schema.bytesEqual) return `Seme.bytesEqual(${projectExpression(reference(field(expression, 0xa0650)), context)}, ${projectExpression(reference(field(expression, 0xa0651)), context)})`;
	if (expression.schema === schema.optionNone) return "Seme.none()";
	if (expression.schema === schema.optionSome) return `Seme.some(${projectExpression(reference(field(expression, 0xa0521)), context)})`;
	if (expression.schema === schema.resultOk) return `Seme.ok(${projectExpression(reference(field(expression, 0x9411)), context)})`;
	if (expression.schema === schema.resultError) return `Seme.error(${projectExpression(reference(field(expression, 0x9421)), context)})`;
	if (expression.schema === schema.variantBindingRead) {
		const binding = context.variants?.get(reference(field(expression, 0xa0610)));
		if (!binding) fail("javascript_projection.variant_binding_scope");
		return binding.name;
	}
	if (expression.schema === schema.optionMatch) {
		const binding = required(context.graph, reference(field(expression, 0xa0632)), schema.variantBinding);
		const name = text(field(binding, 0xa0600));
		if (!/^[A-Za-z_$][\w$]*$/.test(name)) fail("javascript_projection.variant_binding_name");
		return `Seme.matchOption(${projectExpression(reference(field(expression, 0xa0630)), context)}, ${projectMatchCallback(reference(field(expression, 0xa0631)), context, "" )}, ${projectMatchCallback(reference(field(expression, 0xa0633)), { ...context, variants: new Map([...(context.variants || []), [binding.id, { name }]]) }, name)})`;
	}
	if (expression.schema === schema.resultMatch) {
		const ok = required(context.graph, reference(field(expression, 0xa0621)), schema.variantBinding), error = required(context.graph, reference(field(expression, 0xa0623)), schema.variantBinding);
		const okName = text(field(ok, 0xa0600)), errorName = text(field(error, 0xa0600));
		if (![okName, errorName].every((name) => /^[A-Za-z_$][\w$]*$/.test(name)) || okName === errorName) fail("javascript_projection.variant_binding_name");
		return `Seme.matchResult(${projectExpression(reference(field(expression, 0xa0620)), context)}, ${projectMatchCallback(reference(field(expression, 0xa0622)), { ...context, variants: new Map([...(context.variants || []), [ok.id, { name: okName }]]) }, okName)}, ${projectMatchCallback(reference(field(expression, 0xa0624)), { ...context, variants: new Map([...(context.variants || []), [error.id, { name: errorName }]]) }, errorName)})`;
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
	if (expression.schema === schema.collectionLength) {
		return `Seme.length(${projectExpression(reference(field(expression, 0x9f90)), context)})`;
	}
	if (expression.schema === schema.sliceConstruct) {
		required(context.graph, reference(field(expression, 0xa0680)), schema.sliceType);
		return `Seme.slice([${references(field(expression, 0xa0681)).map((value) => projectExpression(value, context)).join(", ")}])`;
	}
	if (expression.schema === schema.dynamicIndexRead) {
		return `Seme.index(${projectExpression(reference(field(expression, 0x9fa0)), context)}, ${projectIndexExpression(reference(field(expression, 0x9fa1)), context)})`;
	}
	if (expression.schema === schema.collectionAppend) {
		return `Seme.append(${projectExpression(reference(field(expression, 0x9fb0)), context)}, ${projectExpression(reference(field(expression, 0x9fb1)), context)})`;
	}
	if (expression.schema === schema.collectionUpdate) {
		return `Seme.update(${projectExpression(reference(field(expression, 0x9fc0)), context)}, ${projectExpression(reference(field(expression, 0x9fc1)), context)}, ${projectExpression(reference(field(expression, 0x9fc2)), context)})`;
	}
	if (expression.schema === schema.sliceRemove) {
		return `Seme.remove(${projectExpression(reference(field(expression, 0xa0660)), context)}, ${projectExpression(reference(field(expression, 0xa0661)), context)})`;
	}
	if (expression.schema === schema.emptyMap) {
		required(context.graph, reference(field(expression, 0xa0410)), schema.mapType);
		return "Seme.emptyMap()";
	}
	if (expression.schema === schema.mapLookup) {
		return `Seme.mapLookupZero(${projectExpression(reference(field(expression, 0xa0420)), context)}, ${projectExpression(reference(field(expression, 0xa0421)), context)})`;
	}
	if (expression.schema === schema.mapLookupOption) {
		const option = required(context.graph, reference(field(expression, 0xa0442)), schema.optionType);
		required(context.graph, reference(field(option, 0xa0500)), schema.integerType);
		return `Seme.mapLookup(${projectExpression(reference(field(expression, 0xa0440)), context)}, ${projectExpression(reference(field(expression, 0xa0441)), context)})`;
	}
	if (expression.schema === schema.mapUpdate) {
		return `Seme.mapInsert(${projectExpression(reference(field(expression, 0xa0430)), context)}, ${projectExpression(reference(field(expression, 0xa0431)), context)}, ${projectExpression(reference(field(expression, 0xa0432)), context)})`;
	}
	if (expression.schema === schema.mapRemove) {
		return `Seme.mapRemove(${projectExpression(reference(field(expression, 0xa0670)), context)}, ${projectExpression(reference(field(expression, 0xa0671)), context)})`;
	}
  if (expression.schema === schema.methodCall) {
    const methodID = reference(field(expression, 0xa0031));
    const method = context.methods.get(methodID);
    if (!method) fail("javascript_projection.method_outside_program");
    const arguments_ = references(field(expression, 0xa0032)).map((argument) => projectExpression(argument, context));
    return `${projectExpression(reference(field(expression, 0xa0030)), context)}.${text(field(method, 0xa0020))}(${arguments_.join(", ")})`;
  }
  if (expression.schema === schema.interfaceValue) {
    const interface_ = context.interfaces.get(reference(field(expression, 0xa0130)));
    const witness = required(context.graph, reference(field(expression, 0xa0132)), schema.satisfactionWitness);
    if (!interface_ || reference(field(witness, 0xa0121)) !== interface_.id) fail("javascript_projection.interface_value");
    return projectExpression(reference(field(expression, 0xa0131)), context);
  }
  if (expression.schema === schema.dynamicMethodCall) {
    const requirementID = reference(field(expression, 0xa0141));
    const interface_ = [...context.interfaces.values()].find((candidate) => candidate.requirements.some((item) => item.id === requirementID));
    const requirement = interface_?.requirements.find((item) => item.id === requirementID);
    if (!requirement) fail("javascript_projection.dynamic_requirement");
    const arguments_ = references(field(expression, 0xa0142)).map((argument) => projectExpression(argument, context));
    return `${projectExpression(reference(field(expression, 0xa0140)), context)}.${requirement.name}(${arguments_.join(", ")})`;
  }
  if (expression.schema === schema.closureConstruct) {
    const functionType = required(context.graph, reference(field(expression, 0xa0230)), schema.functionType);
    const typeParameters = references(field(functionType, 0xa0200));
    const parameterIDs = references(field(expression, 0xa0231));
    if (parameterIDs.length !== typeParameters.length) fail("javascript_projection.closure_arity");
    const parameters = parameterIDs.map((parameterID, index) => {
      const parameter = required(context.graph, parameterID, schema.parameter);
      if (reference(field(parameter, 0x9121)) !== typeParameters[index] || unsigned(field(parameter, 0x9122)) !== BigInt(index)) fail("javascript_projection.closure_parameter");
      return { id: parameterID, name: text(field(parameter, 0x9120)), type: typeName(typeParameters[index], context.graph) };
    });
    const captures = references(field(expression, 0xa0232)).map((captureID) => {
      const capture = required(context.graph, captureID, schema.captureBinding);
      const name = text(field(capture, 0xa0210));
      if (!/^[A-Za-z_$][\w$]*$/.test(name)) fail("javascript_projection.capture_name");
      // Projecting the initializer validates that this binding is available in the outer scope.
      projectExpression(reference(field(capture, 0xa0212)), context);
      return { id: captureID, name, type: typeName(reference(field(capture, 0xa0211)), context.graph) };
    });
    const closureContext = { ...context, parameters: new Map(parameters.map((item) => [item.id, item])), locals: new Map(), captures: new Map(captures.map((item) => [item.id, item])) };
    const body = projectExpression(reference(field(expression, 0xa0233)), closureContext);
    return `(${parameters.map((item) => item.name).join(", ")}) => ${body}`;
  }
  if (expression.schema === schema.indirectCall) {
    const arguments_ = references(field(expression, 0xa0241)).map((argument) => projectExpression(argument, context));
    return `${projectExpression(reference(field(expression, 0xa0240)), context)}(${arguments_.join(", ")})`;
  }
  if (expression.schema === schema.stateTransition) {
    required(context.graph, reference(field(expression, 0xa0050)), schema.transitionType);
    return `{ state: ${projectExpression(reference(field(expression, 0xa0051)), context)}, result: ${projectExpression(reference(field(expression, 0xa0052)), context)} }`;
  }
  if (expression.schema === schema.transitionState) return `${projectExpression(reference(field(expression, 0xa0060)), context)}.state`;
  if (expression.schema === schema.transitionResult) return `${projectExpression(reference(field(expression, 0xa0070)), context)}.result`;
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
    const hasMethods = [...context.methods.values()].some((method) => {
      const receiver = required(context.graph, reference(field(method, 0xa0021)), schema.receiverBinding);
      return reference(field(receiver, 0xa0001)) === record.id;
    });
    if (hasMethods) return `new ${record.name}(${values.map((value) => projectExpression(value, context)).join(", ")})`;
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
      rendered = `Seme.array([${values.map((value) => projectExpression(value, context)).join(", ")}])`;
    } else {
      rendered = projectExpression(collectionID, context);
    }
    return `Seme.index(${rendered}, ${projectExpression(reference(field(expression, 0x9f41)), context)})`;
  }
  const binary = new Map([
	[schema.integerAdd, [0x9140, 0x9141, "+"]],
	[schema.integerMultiply, [0x9900, 0x9901, "*"]],
	[schema.integerSubtract, [0x9a00, 0x9a01, "-"]],
	[schema.integerLessEqual, [0x9160, 0x9161, "<="]],
    [schema.boolAnd, [0x9b10, 0x9b11, "&&"]],
    [schema.boolOr, [0x9c10, 0x9c11, "||"]],
    [schema.stringEqual, [0x9c20, 0x9c21, "==="]],
    [schema.stringConcat, [0x9c30, 0x9c31, "+"]],
  ]).get(expression.schema);
  if (!binary) fail("javascript_projection.unsupported_expression");
  const [leftField, rightField, operator] = binary;
  const rendered = `(${projectExpression(reference(field(expression, leftField)), context)} ${operator} ${projectExpression(reference(field(expression, rightField)), context)})`;
  if (expression.schema === schema.integerAdd || expression.schema === schema.integerMultiply || expression.schema === schema.integerSubtract) {
    return `BigInt.asIntN(64, ${rendered})`;
  }
  return rendered;
}

function projectIndexExpression(id, context) {
	const expression = required(context.graph, id);
	if (expression.schema === schema.integerLiteral) return `${BigInt.asIntN(64, unsigned(field(expression, 0x9700))).toString()}n`;
	if (expression.schema === schema.collectionLength) return `Seme.length(${projectExpression(reference(field(expression, 0x9f90)), context)})`;
	if (expression.schema === schema.integerAdd || expression.schema === schema.integerSubtract) {
		const leftField = expression.schema === schema.integerAdd ? 0x9140 : 0x9a00;
		const rightField = expression.schema === schema.integerAdd ? 0x9141 : 0x9a01;
		const operator = expression.schema === schema.integerAdd ? "+" : "-";
		return `(${projectIndexExpression(reference(field(expression, leftField)), context)} ${operator} ${projectIndexExpression(reference(field(expression, rightField)), context)})`;
	}
	return projectExpression(id, context);
}

function projectMatchBlock(id, context) {
  const block = required(context.graph, id, schema.block);
  const statements = references(field(block, 0x9800));
  if (statements.length !== 1) fail("javascript_projection.match_block_shape");
  const returned = required(context.graph, statements[0], schema.returned);
  const values = references(field(returned, 0x9810));
  if (values.length !== 1) fail("javascript_projection.match_block_shape");
  return projectExpression(values[0], context);
}

function projectMatchCallback(id, context, parameter) {
  const block = required(context.graph, id, schema.block);
  const statements = references(field(block, 0x9800));
  const arguments_ = parameter === "" ? "()" : `(${parameter})`;
  if (statements.length === 1 && required(context.graph, statements[0]).schema === schema.returned) return `${arguments_} => ${projectMatchBlock(id, context)}`;
  return `${arguments_} => {\n${projectBlock(id, context, "  ")}\n}`;
}

function typeName(id, graph) {
  const type = required(graph, id);
  if (type.schema === schema.integerType) return "bigint";
  if (type.schema === schema.stringType) return "string";
  if (type.schema === schema.boolType) return "boolean";
  if (type.schema === schema.bytesType) return "Uint8Array";
  if (type.schema === schema.optionType) return `Seme.Option<${typeName(reference(field(type, 0xa0500)), graph)}>`;
  if (type.schema === schema.resultType) return `Seme.Result<${typeName(reference(field(type, 0x9400)), graph)},${typeName(reference(field(type, 0x9401)), graph)}>`;
  if (type.schema === schema.recordType) return text(field(type, 0x9300));
  if (type.schema === schema.interfaceType) return text(field(type, 0xa0100));
  if (type.schema === schema.functionType) return `function(${references(field(type, 0xa0200)).map((parameter) => typeName(parameter, graph)).join(", ")}): ${typeName(reference(field(type, 0xa0201)), graph)}`;
  if (type.schema === schema.mapType) return `Map<${typeName(reference(field(type, 0xa0400)), graph)},${typeName(reference(field(type, 0xa0401)), graph)}>`;
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
  if (type.schema === schema.transitionType) return `Transition<${typeName(reference(field(type, 0xa0040)), graph)},${typeName(reference(field(type, 0xa0041)), graph)}>`;
  if (type.schema === schema.nativeType) return `Seme.Native<${JSON.stringify(text(field(type, 0xa0710)))},${JSON.stringify(text(field(type, 0xa0711)))}>`;
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
function byteValues(value) {
  const match = /^by ([-0-9a-f]*)$/.exec(atom(value));
  if (!match || (match[1] !== "-" && (match[1].length % 2 !== 0 || !/^[0-9a-f]*$/.test(match[1])))) fail("javascript_projection.invalid_bytes");
  return [...Buffer.from(match[1] === "-" ? "" : match[1], "hex")];
}
function fail(code) { throw new Error(code); }
