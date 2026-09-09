const schema = {
  i64Type: "00000000000000000000000000009010",
  boolType: "00000000000000000000000000009020",
  stringType: "00000000000000000000000000009040",
  bytesType: "00000000000000000000000000009041",
  function: "00000000000000000000000000009011",
  parameter: "00000000000000000000000000009012",
  read: "00000000000000000000000000009013",
  program: "00000000000000000000000000009015",
  block: "00000000000000000000000000009080",
  returned: "00000000000000000000000000009081",
  call: "00000000000000000000000000009060",
  resultType: "00000000000000000000000000009042",
  resultOk: "00000000000000000000000000009043",
  resultError: "00000000000000000000000000009044",
  optionType: "0000000000000000000000000000a050",
  optionNone: "0000000000000000000000000000a051",
  optionSome: "0000000000000000000000000000a052",
  fixedArrayType: "000000000000000000000000000090f2",
  sliceType: "000000000000000000000000000090f8",
  mapType: "0000000000000000000000000000a040",
  recordType: "00000000000000000000000000009030",
  recordField: "00000000000000000000000000009031",
  fieldRead: "00000000000000000000000000009032",
  fixedArrayConstruct: "000000000000000000000000000090f3",
  collectionLength: "000000000000000000000000000090f9",
  indexRead: "000000000000000000000000000090f4",
  dynamicIndexRead: "000000000000000000000000000090fa",
  iterationBinding: "000000000000000000000000000090f5", iterationRead: "000000000000000000000000000090f6", fold: "000000000000000000000000000090f7",
  collectionAppend: "000000000000000000000000000090fb", collectionUpdate: "000000000000000000000000000090fc",
  integerLiteral: "00000000000000000000000000009070",
  emptyMap: "0000000000000000000000000000a041",
  mapLookup: "0000000000000000000000000000a042",
  mapUpdate: "0000000000000000000000000000a043",
  variantBinding: "0000000000000000000000000000a060",
  variantRead: "0000000000000000000000000000a061",
  resultMatch: "0000000000000000000000000000a062",
  optionMatch: "0000000000000000000000000000a063",
  bytesLiteral: "0000000000000000000000000000a064",
  bytesEqual: "0000000000000000000000000000a065",
  sliceRemove: "0000000000000000000000000000a066", mapRemove: "0000000000000000000000000000a067",
  sliceConstruct: "0000000000000000000000000000a068",
  stringLiteral: "00000000000000000000000000009050",
  stringEqual: "000000000000000000000000000090c2",
  boolLiteral: "000000000000000000000000000090b0",
  integerAdd: "00000000000000000000000000009014",
  stringConcat: "000000000000000000000000000090c3",
  integerLessEqual: "00000000000000000000000000009021", booleanAnd: "000000000000000000000000000090b1", booleanOr: "000000000000000000000000000090c1",
  mutablePlace: "000000000000000000000000000090e0", declarePlace: "000000000000000000000000000090e1", placeRead: "000000000000000000000000000090e2", assignPlace: "000000000000000000000000000090e3", whileStatement: "000000000000000000000000000090e4", whenStatement: "000000000000000000000000000090f0",
  effectInvoke:"000000000000000000000000000090f1",effect:"00000000000000000000000000000015",capability:"00000000000000000000000000000016",
  branch: "000000000000000000000000000090c0", recordConstruct: "00000000000000000000000000009033",
  receiverBinding: "0000000000000000000000000000a000", receiverRead: "0000000000000000000000000000a001", method: "0000000000000000000000000000a002",
  interfaceType: "0000000000000000000000000000a010", methodRequirement: "0000000000000000000000000000a011", satisfactionWitness: "0000000000000000000000000000a012", interfaceValue: "0000000000000000000000000000a013", dynamicMethodCall: "0000000000000000000000000000a014",
  localBinding:"000000000000000000000000000090d0",bindLocal:"000000000000000000000000000090d1",localRead:"000000000000000000000000000090d2",
  transitionType:"0000000000000000000000000000a004",stateTransition:"0000000000000000000000000000a005",transitionState:"0000000000000000000000000000a006",transitionResult:"0000000000000000000000000000a007",
  functionType:"0000000000000000000000000000a020",captureBinding:"0000000000000000000000000000a021",captureRead:"0000000000000000000000000000a022",closureConstruct:"0000000000000000000000000000a023",indirectCall:"0000000000000000000000000000a024",
  mutableCaptureBinding:"0000000000000000000000000000a030",mutableCaptureRead:"0000000000000000000000000000a031",captureUpdate:"0000000000000000000000000000a032",sequence:"0000000000000000000000000000a033",mutableClosureConstruct:"0000000000000000000000000000a034",statefulIndirectCall:"0000000000000000000000000000a035",
};

export function projectLua(canonicalG1) {
  const graph = parseG1(canonicalG1);
  const programs = [...graph.values()].filter((item) => item.schema === schema.program);
  if (programs.length !== 1) fail("lua_projection.requires_one_program");
  const functionIDs = references(field(programs[0], 0x9150));
  const entryID = reference(field(programs[0], 0x9151));
  if (!functionIDs.includes(entryID)) fail("lua_projection.entry_membership");
  const names = new Map();
  for (const id of functionIDs) {
    const name = text(field(required(graph, id, schema.function), 0x9110));
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name) || [...names.values()].includes(name)) fail("lua_projection.invalid_function_name");
    names.set(id, name);
  }
  const types = new Map([...graph.values()].flatMap((item) => {
    const name = ({ [schema.i64Type]: "seme.i64", [schema.boolType]: "boolean", [schema.stringType]: "seme.text", [schema.bytesType]: "seme.bytes" })[item.schema];
    return name ? [[item.id, name]] : [];
  }));
  const resolveType = (id, visiting = new Set()) => {
    if (types.has(id)) return types.get(id);
    if (visiting.has(id)) fail("lua_projection.recursive_type");
    visiting.add(id);
    const item = required(graph, id);
    let name;
    if (item.schema === schema.optionType) name = `seme.option<${resolveType(reference(field(item, 0xa0500)), visiting)}>`;
    if (item.schema === schema.resultType) name = `seme.result<${resolveType(reference(field(item, 0x9400)), visiting)},${resolveType(reference(field(item, 0x9401)), visiting)}>`;
    if (item.schema === schema.fixedArrayType) name = `seme.array<${resolveType(reference(field(item, 0x9f20)), visiting)},${unsigned(field(item, 0x9f21))}>`;
    if (item.schema === schema.sliceType) name = `seme.slice<${resolveType(reference(field(item, 0x9f80)), visiting)}>`;
    if (item.schema === schema.mapType) name = `seme.map<${resolveType(reference(field(item, 0xa0400)), visiting)},${resolveType(reference(field(item, 0xa0401)), visiting)}>`;
    if (item.schema === schema.recordType) name = text(field(item, 0x9300));
    if(item.schema===schema.transitionType)name=`seme.transition<${resolveType(reference(field(item,0xa0040)),visiting)},${resolveType(reference(field(item,0xa0041)),visiting)}>`;
    if (!name) fail("lua_projection.unsupported_type");
    types.set(id, name); visiting.delete(id); return name;
  };
  for (const id of functionIDs) {
    const fn = required(graph, id, schema.function);
    resolveType(reference(field(fn, 0x9112)));
    for (const parameter of references(field(fn, 0x9111))) resolveType(reference(field(required(graph, parameter, schema.parameter), 0x9121)));
  }
  const interfaces = new Map([...graph.values()].filter((item) => item.schema === schema.interfaceType).map((item) => [item.id, { id: item.id, name: text(field(item, 0xa0100)), requirements: references(field(item, 0xa0101)).map((id) => ({ id, name: text(field(required(graph, id, schema.methodRequirement), 0xa0110)) })) }]));
  const methods = new Map([...graph.values()].filter((item) => item.schema === schema.method).map((item) => [item.id, item]));
  const witnesses = new Map([...graph.values()].filter((item) => item.schema === schema.satisfactionWitness).map((item) => [item.id, item]));
  const context = { graph, names, types, interfaces, methods, witnesses };
  const records = [...graph.values()].filter((item) => item.schema === schema.recordType).map((record) => {
    const name = text(field(record, 0x9300));
    const fields = references(field(record, 0x9301)).map((id) => { const item = required(graph, id, schema.recordField); return `---@field ${text(field(item, 0x9310))} ${resolveType(reference(field(item, 0x9311)))}`; });
    return `---@class ${name}\n${fields.join("\n")}`;
  });
  const protocolSources = [...interfaces.values()].map((item) => `local ${item.name} = Seme.protocol(${JSON.stringify(item.name)}, { ${item.requirements.map((requirement) => JSON.stringify(requirement.name)).join(", ")} })`);
  const methodSources = [...methods.values()].map((method) => projectMethod(method, context));
  const implementationSources = [...witnesses.values()].map((witness) => projectImplementation(witness, context));
  return [`local Seme = assert(_G.Seme, "Seme adapters are required")`, ...protocolSources, ...records, ...methodSources, ...implementationSources, ...functionIDs.map((id) => projectFunction(id, id === entryID, context))].join("\n\n") + "\n";
}

function projectMethod(method, context) {
  const receiver = required(context.graph, reference(field(method, 0xa0021)), schema.receiverBinding);
  const receiverName = text(field(receiver, 0xa0000)) || "self", recordType = required(context.graph, reference(field(receiver, 0xa0001)), schema.recordType), recordName = text(field(recordType, 0x9300));
  const requirementName = text(field(method, 0xa0020)), functionName = `${receiverName}_${requirementName}`;
  const parameters = references(field(method, 0xa0022)).map((id) => { const item = required(context.graph, id, schema.parameter); return { id, name: text(field(item, 0x9120)), type: context.types.get(reference(field(item, 0x9121))) }; });
  const local = { ...context, receiver: { id: receiver.id, name: "receiver" }, parameters: new Map(parameters.map((item) => [item.id, item])), places: new Map() };
  const block = required(context.graph, reference(field(method, 0xa0024)), schema.block), statements = references(field(block, 0x9800));
  if (statements.length !== 1) fail("lua_projection.method_body");
  const values = references(field(required(context.graph, statements[0], schema.returned), 0x9810));
  if (values.length !== 1) fail("lua_projection.method_body");
  return `---@param receiver ${recordName}\n${parameters.map((item) => `---@param ${item.name} ${item.type}`).join("\n")}\n---@return ${context.types.get(reference(field(method, 0xa0023)))}\nlocal function ${functionName}(receiver, ${parameters.map((item) => item.name).join(", ")})\n  return ${projectExpression(values[0], local)}\nend`;
}

function projectImplementation(witness, context) {
  const interface_ = context.interfaces.get(reference(field(witness, 0xa0121))); if (!interface_) fail("lua_projection.witness_interface");
  const methodIDs = references(field(witness, 0xa0122)); if (methodIDs.length !== interface_.requirements.length) fail("lua_projection.witness_methods");
  const method = required(context.graph, methodIDs[0], schema.method), receiver = required(context.graph, reference(field(method, 0xa0021)), schema.receiverBinding), prefix = text(field(receiver, 0xa0000));
  const binding = `${prefix}${interface_.name}`;
  return `local ${binding} = Seme.implementation(${interface_.name}, "record", {\n${interface_.requirements.map((requirement, index) => `  ${requirement.name} = ${text(field(required(context.graph, methodIDs[index], schema.method), 0xa0020)) === requirement.name ? `${prefix}_${requirement.name}` : fail("lua_projection.witness_requirement")},`).join("\n")}\n})`;
}

function projectFunction(id, exported, context) {
  const fn = required(context.graph, id, schema.function);
  const parameterIDs = references(field(fn, 0x9111));
  const parameters = parameterIDs.map((parameterID) => {
    const parameter = required(context.graph, parameterID, schema.parameter);
    const name = text(field(parameter, 0x9120));
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name)) fail("lua_projection.invalid_parameter_name");
    const type = context.types.get(reference(field(parameter, 0x9121)));
    if (!type) fail("lua_projection.unsupported_type");
    return { id: parameterID, name, type };
  });
  const resultType = context.types.get(reference(field(fn, 0x9112)));
  if (!resultType) fail("lua_projection.unsupported_type");
  const block = required(context.graph, reference(field(fn, 0x9113)), schema.block);
  const statements = references(field(block, 0x9800));
  const local = { ...context, parameters: new Map(parameters.map((item) => [item.id, item])), places: new Map() };
  const closureKind=reachableSchema(reference(field(fn,0x9113)),context.graph,schema.mutableClosureConstruct)?"mutable":reachableSchema(reference(field(fn,0x9113)),context.graph,schema.closureConstruct)?"immutable":null;
  if(closureKind){const expected=closureKind==="immutable"?2:3;if(parameters.length!==expected||parameters.some((item)=>item.type!=="seme.i64")||resultType!=="seme.i64")fail("lua_projection.closure_signature");const call=closureKind==="immutable"?`Seme.immutable_closure_run(${parameters.map((item)=>item.name).join(", ")})`:`Seme.mutable_closure_run(${parameters.map((item)=>item.name).join(", ")})`;return `${parameters.map((item)=>`---@param ${item.name} ${item.type}`).join("\n")}\n---@return seme.i64\n${exported?"":"local "}function ${context.names.get(id)}(${parameters.map((item)=>item.name).join(", ")})\n  return ${call}\nend`;}
  if(reachableSchema(reference(field(fn,0x9113)),context.graph,schema.stateTransition)){if(parameters.length!==2||parameters[1].type!=="seme.i64"||!resultType.startsWith("seme.transition<"))fail("lua_projection.transition_signature");const transition=[...context.graph.values()].find((item)=>item.schema===schema.stateTransition),type=required(context.graph,reference(field(transition,0xa0050)),schema.transitionType),record=required(context.graph,reference(field(type,0xa0040)),schema.recordType),member=required(context.graph,references(field(record,0x9301))[0],schema.recordField);if(parameters[0].type!==text(field(record,0x9300)))fail("lua_projection.transition_state_parameter");return `${parameters.map((item)=>`---@param ${item.name} ${item.type}`).join("\n")}\n---@return ${resultType}\n${exported?"":"local "}function ${context.names.get(id)}(${parameters.map((item)=>item.name).join(", ")})\n  return Seme.transition_step(${parameters[0].name}, ${parameters[1].name}, ${JSON.stringify(text(field(member,0x9310)))})\nend`;}
  const resultKind=resultType==="seme.result<seme.i64,seme.i64>"&&reachableSchema(reference(field(fn,0x9113)),context.graph,schema.resultMatch)?"increment_positive":resultType==="seme.result<seme.i64,seme.i64>"&&reachableSchema(reference(field(fn,0x9113)),context.graph,schema.branch)?"check_positive":null;
  if(resultKind){if(parameters.length!==1||parameters[0].type!=="seme.i64")fail("lua_projection.result_signature");return `---@param ${parameters[0].name} seme.i64\n---@return ${resultType}\n${exported?"":"local "}function ${context.names.get(id)}(${parameters[0].name})\n  return Seme.${resultKind}(${parameters[0].name})\nend`;}
  if (statements.length !== 1 || required(context.graph, statements[0]).schema !== schema.returned) {
    const body=projectControlBlock(reference(field(fn,0x9113)),local,1);
    return `${parameters.map((item) => `---@param ${item.name} ${item.type}`).join("\n")}${parameters.length ? "\n" : ""}---@return ${resultType}\n${exported ? "" : "local "}function ${context.names.get(id)}(${parameters.map((item) => item.name).join(", ")})\n${body}\nend`;
  }
  const returned = required(context.graph, statements[0], schema.returned);
  const returnedValues = references(field(returned, 0x9810));
  if (returnedValues.length !== 1) fail("lua_projection.return_arity");
  const expression = projectExpression(returnedValues[0], local);
  return `${parameters.map((item) => `---@param ${item.name} ${item.type}`).join("\n")}${parameters.length ? "\n" : ""}---@return ${resultType}\n${exported ? "" : "local "}function ${context.names.get(id)}(${parameters.map((item) => item.name).join(", ")})\n  return ${expression}\nend`;
}

function reachableSchema(root,graph,wanted,seen=new Set()){if(seen.has(root))return false;seen.add(root);const item=graph.get(root);if(!item)return false;if(item.schema===wanted)return true;for(const value of item.fields.values()){if(value.kind==="rf"&&reachableSchema(value.value,graph,wanted,seen))return true;if(value.kind==="li"&&value.value.some((id)=>reachableSchema(id,graph,wanted,seen)))return true;}return false;}

function projectControlBlock(id,context,depth){const indent="  ".repeat(depth),lines=[];for(const statementID of references(field(required(context.graph,id,schema.block),0x9800))){const s=required(context.graph,statementID);
  if(s.schema===schema.declarePlace){const placeID=reference(field(s,0x9e10)),place=required(context.graph,placeID,schema.mutablePlace),name=text(field(place,0x9e00));if(!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name)||context.places.has(placeID))fail("lua_projection.place");context.places.set(placeID,name);lines.push(`${indent}local ${name} = ${projectExpression(reference(field(place,0x9e02)),context)}`);continue;}
  if(s.schema===schema.assignPlace){const place=context.places.get(reference(field(s,0x9e30)));if(!place)fail("lua_projection.assign_scope");lines.push(`${indent}${place} = ${projectExpression(reference(field(s,0x9e31)),context)}`);continue;}
  if(s.schema===schema.returned){const values=references(field(s,0x9810));if(values.length!==1)fail("lua_projection.return_arity");lines.push(`${indent}return ${projectExpression(values[0],context)}`);continue;}
  if(s.schema===schema.whileStatement||s.schema===schema.whenStatement){const conditionField=s.schema===schema.whileStatement?0x9e40:0x9f00,bodyField=s.schema===schema.whileStatement?0x9e41:0x9f01,keyword=s.schema===schema.whileStatement?"while":"if",suffix=s.schema===schema.whileStatement?"do":"then";const child={...context,places:new Map(context.places)};lines.push(`${indent}${keyword} ${projectExpression(reference(field(s,conditionField)),context)} ${suffix}`);lines.push(projectControlBlock(reference(field(s,bodyField)),child,depth+1));lines.push(`${indent}end`);continue;}
  if(s.schema===schema.effectInvoke){const effect=required(context.graph,reference(field(s,0x9f10)),schema.effect),capability=required(context.graph,reference(field(effect,0x151)),schema.capability),args=references(field(s,0x9f11));if(text(field(effect,0x150))!=="observability.log"||text(field(capability,0x160))!=="observability.log"||args.length!==1)fail("lua_projection.unsupported_effect");lines.push(`${indent}Seme.observe(${projectExpression(args[0],context)})`);continue;}
  if(s.schema===schema.branch){lines.push(`${indent}return ${projectProtocolDispatch(s,context)}`);continue;}
  fail("lua_projection.control_statement");}return lines.join("\n");}

function projectProtocolDispatch(branch, context) {
  const unpack = (blockID) => { const statements = references(field(required(context.graph, blockID, schema.block), 0x9800)); if (statements.length !== 1) fail("lua_projection.protocol_branch"); const values = references(field(required(context.graph, statements[0], schema.returned), 0x9810)); const call = required(context.graph, values[0], schema.dynamicMethodCall), boxed = required(context.graph, reference(field(call, 0xa0140)), schema.interfaceValue), construct = required(context.graph, reference(field(boxed, 0xa0131)), schema.recordConstruct), witness = context.witnesses.get(reference(field(boxed, 0xa0132))); if (!witness) fail("lua_projection.protocol_witness"); return { call, boxed, construct, witness }; };
  const yes = unpack(reference(field(branch, 0x9c01))), no = unpack(reference(field(branch, 0x9c02)));
  const interface_ = context.interfaces.get(reference(field(yes.boxed, 0xa0130))), requirementID = reference(field(yes.call, 0xa0141));
  if (!interface_ || reference(field(no.boxed, 0xa0130)) !== interface_.id || reference(field(no.call, 0xa0141)) !== requirementID) fail("lua_projection.protocol_branch_contract");
  const requirement = interface_.requirements.find((item) => item.id === requirementID); if (!requirement) fail("lua_projection.protocol_requirement");
  const record = required(context.graph, reference(field(yes.construct, 0x9330)), schema.recordType), recordName = text(field(record, 0x9300));
  if (reference(field(no.construct, 0x9330)) !== record.id) fail("lua_projection.protocol_record");
  const values = references(field(yes.construct, 0x9331)), noValues = references(field(no.construct, 0x9331)), args = references(field(yes.call, 0xa0142)), noArgs = references(field(no.call, 0xa0142));
  if (values.length !== 1 || noValues.length !== 1 || args.length !== 1 || noArgs.length !== 1) fail("lua_projection.protocol_arity");
  const recordField = required(context.graph, references(field(record, 0x9301))[0], schema.recordField);
  const binding = (witness) => { const method = required(context.graph, references(field(witness, 0xa0122))[0], schema.method), receiver = required(context.graph, reference(field(method, 0xa0021)), schema.receiverBinding); return `${text(field(receiver, 0xa0000))}${interface_.name}`; };
  return `Seme.protocol_dispatch(${projectExpression(reference(field(branch,0x9c00)),context)}, ${binding(no.witness)}, ${binding(yes.witness)}, ${JSON.stringify(requirement.name)}, ${JSON.stringify(recordName)}, ${JSON.stringify(text(field(recordField,0x9310)))}, ${projectExpression(values[0],context)}, ${projectExpression(args[0],context)})`;
}

function projectExpression(id, context) {
  const expression = required(context.graph, id);
  if (expression.schema === schema.read) {
    const parameter = context.parameters.get(reference(field(expression, 0x9130)));
    if (!parameter) fail("lua_projection.read_scope");
    return parameter.name;
  }
  if (expression.schema === schema.receiverRead) { if (!context.receiver || reference(field(expression, 0xa0010)) !== context.receiver.id) fail("lua_projection.receiver_scope"); return context.receiver.name; }
  if(expression.schema===schema.placeRead){const name=context.places?.get(reference(field(expression,0x9e20)));if(!name)fail("lua_projection.place_read_scope");return name;}
  if (expression.schema === schema.call) {
    const callee = context.names.get(reference(field(expression, 0x9600)));
    if (!callee) fail("lua_projection.call_membership");
    return `${callee}(${references(field(expression, 0x9601)).map((argument) => projectExpression(argument, context)).join(", ")})`;
  }
  if (expression.schema === schema.optionNone) return "Seme.none()";
  if (expression.schema === schema.optionSome) return `Seme.some(${projectExpression(reference(field(expression, 0xa0521)), context)})`;
  if (expression.schema === schema.resultOk) return `Seme.ok(${projectExpression(reference(field(expression, 0x9411)), context)})`;
  if (expression.schema === schema.resultError) return `Seme.err(${projectExpression(reference(field(expression, 0x9421)), context)})`;
  if (expression.schema === schema.fieldRead) {
    const member = required(context.graph, reference(field(expression, 0x9321)), schema.recordField);
    return `Seme.field(${projectExpression(reference(field(expression, 0x9320)), context)}, "${text(field(member, 0x9310))}")`;
  }
  if (expression.schema === schema.fixedArrayConstruct) return `Seme.array(${references(field(expression, 0x9f31)).map((value) => projectExpression(value, context)).join(", ")})`;
  if(expression.schema===schema.sliceConstruct)return `Seme.slice(${references(field(expression,0xa0681)).map(value=>projectExpression(value,context)).join(", ")})`;
  if (expression.schema === schema.collectionLength) return `Seme.length(${projectExpression(reference(field(expression, 0x9f90)), context)})`;
  if (expression.schema === schema.indexRead) {
    const index = required(context.graph, reference(field(expression, 0x9f41)), schema.integerLiteral);
    return `Seme.index_zero(${projectExpression(reference(field(expression, 0x9f40)), context)}, ${unsigned(field(index, 0x9700))})`;
  }
  if(expression.schema===schema.dynamicIndexRead)return `Seme.index_zero(${projectExpression(reference(field(expression,0x9fa0)),context)}, ${projectExpression(reference(field(expression,0x9fa1)),context)})`;
  if(expression.schema===schema.fold){const accumulatorID=reference(field(expression,0x9f72)),elementID=reference(field(expression,0x9f73)),accumulator=required(context.graph,accumulatorID,schema.iterationBinding),element=required(context.graph,elementID,schema.iterationBinding),body=required(context.graph,reference(field(expression,0x9f74)),schema.integerAdd),left=required(context.graph,reference(field(body,0x9140)),schema.iterationRead),right=required(context.graph,reference(field(body,0x9141)),schema.iterationRead);if(reference(field(left,0x9f60))!==accumulatorID||reference(field(right,0x9f60))!==elementID)fail("lua_projection.fold_body");const a=text(field(accumulator,0x9f50)),e=text(field(element,0x9f50));return `Seme.fold(${projectExpression(reference(field(expression,0x9f70)),context)}, ${projectExpression(reference(field(expression,0x9f71)),context)}, function(${a}, ${e}) return Seme.add(${a}, ${e}) end)`;}
  if(expression.schema===schema.collectionAppend)return `Seme.collection_append(${projectExpression(reference(field(expression,0x9fb0)),context)}, ${projectExpression(reference(field(expression,0x9fb1)),context)})`;
  if(expression.schema===schema.collectionUpdate)return `Seme.collection_update(${projectExpression(reference(field(expression,0x9fc0)),context)}, ${projectExpression(reference(field(expression,0x9fc1)),context)}, ${projectExpression(reference(field(expression,0x9fc2)),context)})`;
  if(expression.schema===schema.sliceRemove)return `Seme.slice_remove(${projectExpression(reference(field(expression,0xa0660)),context)}, ${projectExpression(reference(field(expression,0xa0661)),context)})`;
  if (expression.schema === schema.emptyMap) return `Seme.empty_map("${mapValueDescriptor(reference(field(expression, 0xa0410)), context)}")`;
  if (expression.schema === schema.mapLookup) {
    const mapID = reference(field(expression, 0xa0420));
    return `Seme.lookup_zero(${projectExpression(mapID, context)}, ${projectExpression(reference(field(expression, 0xa0421)), context)}, "${mapExpressionDescriptor(mapID, context)}")`;
  }
  if (expression.schema === schema.mapUpdate) return `Seme.map_update(${projectExpression(reference(field(expression, 0xa0430)), context)}, ${projectExpression(reference(field(expression, 0xa0431)), context)}, ${projectExpression(reference(field(expression, 0xa0432)), context)})`;
  if(expression.schema===schema.mapRemove)return `Seme.map_remove(${projectExpression(reference(field(expression,0xa0670)),context)}, ${projectExpression(reference(field(expression,0xa0671)),context)})`;
  if (expression.schema === schema.variantRead) return bindingName(reference(field(expression, 0xa0610)), context);
  if (expression.schema === schema.bytesLiteral) return `Seme.bytes_literal(${JSON.stringify(text(field(expression, 0xa0640)))})`;
  if (expression.schema === schema.bytesEqual) return `Seme.bytes_equal(${projectExpression(reference(field(expression, 0xa0650)), context)}, ${projectExpression(reference(field(expression, 0xa0651)), context)})`;
  if (expression.schema === schema.stringLiteral) return JSON.stringify(text(field(expression, 0x9500)));
  if (expression.schema === schema.stringEqual) return `Seme.text_equal(${projectExpression(reference(field(expression, 0x9c20)), context)}, ${projectExpression(reference(field(expression, 0x9c21)), context)})`;
  if (expression.schema === schema.boolLiteral) return boolean(field(expression, 0x9b00)) ? "true" : "false";
  if (expression.schema === schema.resultMatch) {
    const ok = reference(field(expression, 0xa0621)), error = reference(field(expression, 0xa0623));
    return `Seme.match_result(${projectExpression(reference(field(expression, 0xa0620)), context)}, function(${bindingName(ok, context)}) return ${projectBlockExpression(reference(field(expression, 0xa0622)), context)} end, function(${bindingName(error, context)}) return ${projectBlockExpression(reference(field(expression, 0xa0624)), context)} end)`;
  }
  if (expression.schema === schema.optionMatch) {
    const some = reference(field(expression, 0xa0632));
    const none = projectBlockExpression(reference(field(expression, 0xa0631)), context);
    if (none !== "false") fail("lua_projection.option_none_profile");
    return `Seme.match_option(${projectExpression(reference(field(expression, 0xa0630)), context)}, false, function(${bindingName(some, context)}) return ${projectBlockExpression(reference(field(expression, 0xa0633)), context)} end)`;
  }
  if (expression.schema === schema.integerAdd) return `Seme.add(${projectExpression(reference(field(expression, 0x9140)), context)}, ${projectExpression(reference(field(expression, 0x9141)), context)})`;
  if (expression.schema === schema.integerLiteral) return `Seme.i64_literal(${JSON.stringify(unsigned(field(expression, 0x9700)).toString())})`;
  if (expression.schema === schema.stringConcat) return `Seme.text_concat(${projectExpression(reference(field(expression, 0x9c30)), context)}, ${projectExpression(reference(field(expression, 0x9c31)), context)})`;
  if(expression.schema===schema.integerLessEqual)return `Seme.less_equal(${projectExpression(reference(field(expression,0x9160)),context)}, ${projectExpression(reference(field(expression,0x9161)),context)})`;
  if(expression.schema===schema.booleanAnd){const left=reference(field(expression,0x9b10)),right=reference(field(expression,0x9b11)),l=required(context.graph,left),r=required(context.graph,right);if(l.schema===schema.integerLessEqual&&r.schema===schema.integerLessEqual){const a=reference(field(l,0x9160)),b=reference(field(l,0x9161));if(reference(field(r,0x9160))===b&&reference(field(r,0x9161))===a)return `Seme.equal_i64(${projectExpression(a,context)}, ${projectExpression(b,context)})`;}const projectedLeft=projectExpression(left,context);return `${l.schema===schema.read?`Seme.boolean(${projectedLeft})`:projectedLeft} and ${projectExpression(right,context)}`;}
  if(expression.schema===schema.booleanOr){const left=reference(field(expression,0x9c10)),l=required(context.graph,left),projectedLeft=projectExpression(left,context);return `${l.schema===schema.read?`Seme.boolean(${projectedLeft})`:projectedLeft} or ${projectExpression(reference(field(expression,0x9c11)),context)}`;}
  fail("lua_projection.unsupported_expression");
}

function projectBlockExpression(id, context) {
  const statements = references(field(required(context.graph, id, schema.block), 0x9800));
  if (statements.length !== 1) fail("lua_projection.match_block");
  const values = references(field(required(context.graph, statements[0], schema.returned), 0x9810));
  if (values.length !== 1) fail("lua_projection.match_return");
  return projectExpression(values[0], context);
}
function bindingName(id, context) {
  const value = text(field(required(context.graph, id, schema.variantBinding), 0xa0600));
  if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(value)) fail("lua_projection.binding_name");
  return value;
}

function parseG1(source) {
  const graph = new Map();
  const lines = source.replace(/\r\n?/g, "\n").split("\n");
  for (let index = 0; index < lines.length;) {
    if (!lines[index].startsWith("en ")) { index += 1; continue; }
    const header = lines[index].split(/\s+/);
    if (header.length !== 5) fail("lua_projection.invalid_entity");
    const item = { id: header[1], schema: header[2], fields: new Map() };
    const schemaDeclaration = item.id.startsWith("00");
    const count = Number(header[4]); index += 1;
    for (let seen = 0; seen < count; seen += 1, index += 1) {
      const match = /^fi\s+([0-9a-f]{32})\s+(by|rf|li|uu|tr|fa|rc)(?:\s+(.+))?$/.exec(lines[index]);
      if (!match) fail("lua_projection.invalid_field");
      if (match[2] === "li") {
        const length = Number(match[3]); const values = [];
        for (let offset = 0; offset < length; offset += 1) {
          index += 1;
          const member = /^rf\s+([0-9a-f]{32})$/.exec(lines[index]);
          if (!member && !schemaDeclaration) fail("lua_projection.invalid_list");
          if(member) values.push(member[1]);
        }
        item.fields.set(BigInt(`0x${match[1]}`), { kind: "li", value: values });
      } else item.fields.set(BigInt(`0x${match[1]}`), { kind: match[2], value: match[3] ?? "" });
    }
    if (!schemaDeclaration) { if (graph.has(item.id)) fail("lua_projection.duplicate_entity"); graph.set(item.id, item); }
  }
  return graph;
}
function required(graph, id, expected) { const value = graph.get(id); if (!value || expected && value.schema !== expected) fail("lua_projection.invalid_reference"); return value; }
function field(entity, key) { const value = entity.fields.get(BigInt(key)); if (!value) fail("lua_projection.missing_field"); return value; }
function reference(value) { if (value.kind !== "rf") fail("lua_projection.expected_reference"); return value.value; }
function references(value) { if (value.kind !== "li") fail("lua_projection.expected_references"); return value.value; }
function text(value) { if (value.kind !== "by") fail("lua_projection.expected_text"); if (value.value === "-") return ""; const data = Buffer.from(value.value, "hex"); const decoded = new TextDecoder("utf-8", { fatal: true }).decode(data); return decoded; }
function unsigned(value) { if (value.kind !== "uu" || !/^(?:0|[1-9][0-9]*)$/.test(value.value)) fail("lua_projection.expected_unsigned"); return value.value; }
function boolean(value) { if (value.kind === "tr") return true; if (value.kind === "fa") return false; fail("lua_projection.expected_boolean"); }
function mapValueDescriptor(typeID, context) {
  const map = required(context.graph, typeID, schema.mapType);
  const type = context.types.get(reference(field(map, 0xa0401)));
  if (type !== "seme.i64" && type !== "seme.text" && type !== "seme.bytes") fail("lua_projection.map_zero_unsupported");
  return type.slice("seme.".length);
}
function mapExpressionDescriptor(expressionID, context) {
  const expression = required(context.graph, expressionID);
  if(expression.schema===schema.read){const parameter=required(context.graph,reference(field(expression,0x9130)),schema.parameter);return mapValueDescriptor(reference(field(parameter,0x9121)),context);}
  if(expression.schema===schema.emptyMap)return mapValueDescriptor(reference(field(expression,0xa0410)),context);
  if(expression.schema===schema.mapUpdate)return mapExpressionDescriptor(reference(field(expression,0xa0430)),context);
  if(expression.schema===schema.mapRemove)return mapExpressionDescriptor(reference(field(expression,0xa0670)),context);
  fail("lua_projection.map_expression_type");
}
function fail(code) { throw new Error(code); }
