import crypto from "node:crypto";
import { parseProtocolDeclarations, stripProtocolDeclarations } from "./lua-protocol-parser.mjs";
import { projectLua } from "./lua-projector.mjs";

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
  collectionAppend: "000000000000000000000000000090fb",
  collectionUpdate: "000000000000000000000000000090fc",
  integerLiteral: "00000000000000000000000000009070",
  emptyMap: "0000000000000000000000000000a041",
  mapLookup: "0000000000000000000000000000a042",
  mapLookupOption: "0000000000000000000000000000a044",
  mapUpdate: "0000000000000000000000000000a043",
  variantBinding: "0000000000000000000000000000a060",
  variantRead: "0000000000000000000000000000a061",
  resultMatch: "0000000000000000000000000000a062",
  optionMatch: "0000000000000000000000000000a063",
  bytesLiteral: "0000000000000000000000000000a064",
  bytesEqual: "0000000000000000000000000000a065",
  sliceRemove: "0000000000000000000000000000a066",
  mapRemove: "0000000000000000000000000000a067",
  sliceConstruct: "0000000000000000000000000000a068",
  stringLiteral: "00000000000000000000000000009050",
  stringEqual: "000000000000000000000000000090c2",
  boolLiteral: "000000000000000000000000000090b0",
  integerAdd: "00000000000000000000000000009014",
  integerMultiply: "00000000000000000000000000009090",
  integerSubtract:"000000000000000000000000000090a0",
  stringConcat: "000000000000000000000000000090c3",
  integerLessEqual: "00000000000000000000000000009021",
  booleanAnd: "000000000000000000000000000090b1",
  booleanOr: "000000000000000000000000000090c1",
  mutablePlace: "000000000000000000000000000090e0",
  declarePlace: "000000000000000000000000000090e1",
  placeRead: "000000000000000000000000000090e2",
  assignPlace: "000000000000000000000000000090e3",
  whileStatement: "000000000000000000000000000090e4",
  whenStatement: "000000000000000000000000000090f0",
  effectInvoke:"000000000000000000000000000090f1", effect:"00000000000000000000000000000015", capability:"00000000000000000000000000000016",
  branch: "000000000000000000000000000090c0",
  recordConstruct: "00000000000000000000000000009033",
  receiverBinding: "0000000000000000000000000000a000",
  receiverRead: "0000000000000000000000000000a001",
  method: "0000000000000000000000000000a002",
  interfaceType: "0000000000000000000000000000a010",
  methodRequirement: "0000000000000000000000000000a011",
  satisfactionWitness: "0000000000000000000000000000a012",
  interfaceValue: "0000000000000000000000000000a013",
  dynamicMethodCall: "0000000000000000000000000000a014",
  localBinding: "000000000000000000000000000090d0", bindLocal: "000000000000000000000000000090d1", localRead: "000000000000000000000000000090d2",
  transitionType: "0000000000000000000000000000a004", stateTransition:"0000000000000000000000000000a005", transitionState: "0000000000000000000000000000a006", transitionResult: "0000000000000000000000000000a007",
  functionType: "0000000000000000000000000000a020", captureBinding: "0000000000000000000000000000a021", captureRead: "0000000000000000000000000000a022", closureConstruct: "0000000000000000000000000000a023", indirectCall: "0000000000000000000000000000a024",
  mutableCaptureBinding: "0000000000000000000000000000a030", mutableCaptureRead: "0000000000000000000000000000a031", captureUpdate: "0000000000000000000000000000a032", sequence: "0000000000000000000000000000a033", mutableClosureConstruct: "0000000000000000000000000000a034", statefulIndirectCall: "0000000000000000000000000000a035",
};

/**
 * Lift the deliberately bounded Lua 5.1 / LuaJIT profile directly to Seme.
 * `sources` is an ordered array of { name, source }; file names never affect
 * semantic identity.
 */
export function liftLua({ sources, packagePath, revision, moduleG1, entryName }) {
  if (!Array.isArray(sources) || sources.length === 0) fail("lua.requires_sources");
  if (!packagePath || !Number.isSafeInteger(revision) || revision < 1) fail("lua.invalid_snapshot");
  const projected = verifyProjectionEnvelope(sources);
  if (projected) return projected;
  const records = parseRecords(sources, packagePath);
  const protocolUnits = sources.map(({ name, source }) => parseProtocolDeclarations(source, name));
  const protocolBindings = new Map(), implementationBindings = new Map();
  for (const unit of protocolUnits) {
    for (const [name, value] of unit.protocols) { if (protocolBindings.has(name)) fail("lua.duplicate_protocol_binding"); protocolBindings.set(name, value); }
    for (const [name, value] of unit.implementations) { if (implementationBindings.has(name)) fail("lua.duplicate_implementation_binding"); implementationBindings.set(name, value); }
  }
  const protocolTypes=new Map([...protocolBindings.values()].map(item=>[item.name,{id:stableID("execution","interface",packagePath,item.name),name:item.name}]));
  const declarations = sources.flatMap(({ name, source }) => parseSource(normalizeModuleSyntax(stripProtocolDeclarations(source), name), name, records,protocolTypes));
  const byName = new Map();
  for (const declaration of declarations) {
    if (byName.has(declaration.name)) fail("lua.duplicate_function", declaration.location);
    byName.set(declaration.name, declaration);
  }
  for (const implementation of implementationBindings.values()) for (const method of implementation.methods.values()) {
    if (!byName.has(method)) fail("lua.implementation_method_scope");
  }
  const entry = entryName ? byName.get(entryName) : declarations.find((item) => item.exported);
  if (!entry) fail("lua.entry_not_found");
  if (!entry.exported) fail("lua.entry_must_be_global", entry.location);

  const additions = [];
  for (const type of declarations.flatMap((item) => [...item.parameters.map((parameter) => parameter.type), item.resultType])) ensureType(type, additions, records);
  const methodNames = new Set([...implementationBindings.values()].flatMap((item) => [...item.methods.values()]));
  const descriptions = declarations.map((declaration) => ({
    ...declaration,
    id: declaration.identity ?? stableID("session-declaration", packagePath, declaration.name),
  }));
  const identities = new Set();
  for (const description of descriptions) {
    if (identities.has(description.id)) fail("lua.duplicate_semantic_identity", description.location);
    identities.add(description.id);
  }
  const descriptionsByName = new Map(descriptions.map((item) => [item.name, item]));

  const protocols = new Map();
  for (const protocol of protocolBindings.values()) {
    const requirements = protocol.requirements.map((name) => {
      const implementations = [...implementationBindings.values()].filter((item) => item.protocol === protocol.binding);
      const signatures = implementations.map((item) => descriptionsByName.get(item.methods.get(name)));
      if (!signatures.length || signatures.some((item) => !item || item.parameters.length !== 2 || !(item.parameters[0].type.startsWith("record:") || item.parameters[0].type === "i64") || item.parameters[1].type !== "i64" || item.resultType !== "i64")) fail("lua.protocol_method_signature");
      const id = stableID("execution", "requirement", packagePath, protocol.name, name);
      additions.push(graphEntity(id, entity(id, schema.methodRequirement, [[0xa0110, bytes(name)], [0xa0111, refs([ids.i64])], [0xa0112, ref(ids.i64)]])));
      return { name, id };
    });
    const id = stableID("execution", "interface", packagePath, protocol.name);
    additions.push(graphEntity(id, entity(id, schema.interfaceType, [[0xa0100, bytes(protocol.name)], [0xa0101, refs(requirements.map((item) => item.id))]])));
    protocols.set(protocol.binding, { ...protocol, id, requirements });
  }
  const implementations = new Map();
  for (const implementation of implementationBindings.values()) {
    const protocol = protocols.get(implementation.protocol);
    const prefix = implementation.binding.endsWith(protocol.name) ? implementation.binding.slice(0, -protocol.name.length) : implementation.binding;
    const methodIDs = [];
    let concreteID;
    for (const requirement of protocol.requirements) {
      const description = descriptionsByName.get(implementation.methods.get(requirement.name));
      const receiverType = description.parameters[0].type;
      if (implementation.concreteKind !== (receiverType.startsWith("record:") ? "record" : receiverType)) fail("lua.implementation_concrete_kind");
      concreteID ??= typeID(receiverType);
      if (concreteID !== typeID(receiverType)) fail("lua.implementation_receiver_kind");
      const receiverID = implementation.receiverID??stableID("execution", "receiver", packagePath, implementation.binding);
      if (!additions.some((item) => item.id === receiverID)) additions.push(graphEntity(receiverID, entity(receiverID, schema.receiverBinding, [[0xa0000, bytes(implementation.receiverName??(prefix || "self"))], [0xa0001, ref(concreteID)]])));
      const parameter = description.parameters[1], parameterID = stableID("execution", description.id, "parameter", "0");
      additions.push(graphEntity(parameterID, entity(parameterID, schema.parameter, [[0x9120, bytes(parameter.name)], [0x9121, ref(ids.i64)], [0x9122, "uu 0"]])));
      const methodContext = { description: { ...description, parameters: [parameter] }, parameterIDs: [parameterID], descriptionsByName, additions, records, receiver: { id: receiverID, name: description.parameters[0].name, type: receiverType } };
      const expression = emitExpression(description.expression, methodContext, "body.statement.expression");
      const returnedID = stableID("execution", description.id, "body.statement", "return"), blockID = stableID("execution", description.id, "body", "block");
      additions.push(graphEntity(returnedID, entity(returnedID, schema.returned, [[0x9810, refs([expression])]])), graphEntity(blockID, entity(blockID, schema.block, [[0x9800, refs([returnedID])]])));
      additions.push(graphEntity(description.id, entity(description.id, schema.method, [[0xa0020, bytes(requirement.name)], [0xa0021, ref(receiverID)], [0xa0022, refs([parameterID])], [0xa0023, ref(ids.i64)], [0xa0024, ref(blockID)]])));
      methodIDs.push(description.id);
    }
    const witnessID = implementation.witnessID??stableID("execution", "witness", packagePath, implementation.binding);
    additions.push(graphEntity(witnessID, entity(witnessID, schema.satisfactionWitness, [[0xa0120, ref(concreteID)], [0xa0121, ref(protocol.id)], [0xa0122, refs(methodIDs)]])));
    implementations.set(implementation.binding, { ...implementation, protocol, prefix, concreteID, witnessID });
  }

  for (const description of descriptions.filter((item) => !methodNames.has(item.name))) {
    const parameterIDs = description.parameters.map((parameter, index) => {
      const id = stableID("execution", description.id, "parameter", String(index));
      additions.push(graphEntity(id, entity(id, schema.parameter, [
        [0x9120, bytes(parameter.name)], [0x9121, ref(typeID(parameter.type))], [0x9122, `uu ${index}`],
      ])));
      return id;
    });
    const context = { description, parameterIDs, descriptionsByName, additions, records, implementations };
    let blockID;
    if (description.statements) blockID = emitControlBlock(description.statements, { ...context, symbols: new Map(description.parameters.map((p, i) => [p.name, { kind: "parameter", type: p.type, id: parameterIDs[i] }])) }, "body", true);
    else if (description.expression.kind === "protocol_dispatch" || description.expression.kind === "protocol_dispatch_value") blockID = emitProtocolDispatch(description.expression, { ...context, implementations }, "body");
    else if (description.expression.kind === "immutable_closure_run") blockID = emitImmutableClosureRun(description.expression, context, "body");
    else if (description.expression.kind === "mutable_closure_run") blockID = emitMutableClosureRun(description.expression, context, "body");
    else if (description.expression.kind === "transition_step") blockID = emitTransitionStep(description.expression, context, "body");
    else if (description.expression.kind === "check_positive") blockID = emitCheckPositive(description.expression, context, "body");
    else if (description.expression.kind === "increment_positive") blockID = emitIncrementPositive(description.expression, context, "body");
    else {
      const expression = emitExpression(description.expression, context, "body.statement.expression");
      const returnedID = stableID("execution", description.id, "body.statement", "return");
      blockID = stableID("execution", description.id, "body", "block");
      additions.push(graphEntity(returnedID, entity(returnedID, schema.returned, [[0x9810, refs([expression])]])));
      additions.push(graphEntity(blockID, entity(blockID, schema.block, [[0x9800, refs([returnedID])]])));
    }
    additions.push(graphEntity(description.id, entity(description.id, schema.function, [
      [0x9110, bytes(description.name)], [0x9111, refs(parameterIDs)], [0x9112, ref(typeID(description.resultType))], [0x9113, ref(blockID)],
    ])));
  }
  const functionIDs = descriptions.filter((item) => !methodNames.has(item.name)).map((item) => item.id).sort();
  const programID = stableID("session-program", packagePath);
  additions.push(graphEntity(programID, entity(programID, schema.program, [
    [0x9150, refs(functionIDs)], [0x9151, ref(descriptionsByName.get(entry.name).id)],
  ])));
  return compose(moduleG1, stableID("session-revision", packagePath, String(revision)), additions);
}

// Lua modules are provider mechanics, not Core semantics.  The bounded project
// provider resolves static require bindings before the ordinary expression
// lifter sees calls, while Package-v2 retains the original module boundary.
function normalizeModuleSyntax(source, file) {
  const aliases = new Map();
  const exported = new Set();
  for (const table of source.matchAll(/^\s*return\s*\{([^}]*)\}\s*$/gm)) for (const item of table[1].split(",").map(value => value.trim()).filter(Boolean)) {
    const binding = /^([A-Za-z_]\w*)\s*=\s*([A-Za-z_]\w*)$/.exec(item);
    if (!binding || binding[1] !== binding[2] || exported.has(binding[1])) fail("lua.module_export_shape", { file, line: source.slice(0, table.index).split(/\r?\n/).length, column: 1 });
    exported.add(binding[1]);
  }
  let normalized = source.replace(/^(\s*)local\s+([A-Za-z_]\w*)\s*=\s*require\s*\(\s*["']([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)["']\s*\)\s*$/gm,
    (whole, indent, alias, requested) => {
      if (aliases.has(alias)) fail("lua.duplicate_require_alias", { file, line: source.slice(0, source.indexOf(whole)).split(/\r?\n/).length, column: 1 });
      aliases.set(alias, requested);
      return "";
    });
  normalized = normalized.replace(/^\s*return\s*\{\s*(?:[A-Za-z_]\w*\s*=\s*[A-Za-z_]\w*\s*,?\s*)+\}\s*$/gm, "");
  for (const alias of aliases.keys()) normalized = normalized.replace(new RegExp(`\\b${alias}\\.([A-Za-z_]\\w*)`, "g"), "$1");
  for (const name of exported) normalized = normalized.replace(new RegExp(`^(\\s*)local\\s+function\\s+${name}\\b`, "m"), `$1function ${name}`);
  return normalized;
}

function verifyProjectionEnvelope(sources) {
  const marked = sources.filter(({ source }) => /^-- seme:(?:projection-v1|graph) /m.test(source));
  if (marked.length === 0) return null;
  if (sources.length !== 1 || marked.length !== 1) fail("lua_projection.envelope_source_count");
  const source = marked[0].source;
  const digests = [...source.matchAll(/^-- seme:projection-v1 ([^\s]+)\s*$/gm)].map((match) => match[1]);
  const chunks = [...source.matchAll(/^-- seme:graph ([^\s]+)\s*$/gm)].map((match) => match[1]);
  if (digests.length !== 1 || chunks.length === 0) fail("lua_projection.envelope_incomplete");
  let graph;
  try { graph = Buffer.from(chunks.join(""), "base64url").toString("utf8"); }
  catch { fail("lua_projection.envelope_encoding"); }
  if (crypto.createHash("sha256").update(graph).digest("hex") !== digests[0]) fail("lua_projection.envelope_digest");
  let reprojected;
  try { reprojected = projectLua(graph); }
  catch { fail("lua_projection.envelope_graph"); }
  if (reprojected !== source) fail("lua_projection.envelope_source_mismatch");
  return graph;
}

function emitControlBlock(statements, context, path, requireReturn = false) {
  const emitted = []; let terminal = false;
  for (let index = 0; index < statements.length; index += 1) {
    const statement = statements[index], statementPath = `${path}.statement.${index}`;
    if (terminal) fail("lua.unreachable_statement", statement.location);
    if (statement.kind === "declare") {
      if (context.symbols.has(statement.name)) fail("lua.duplicate_local", statement.location);
      if(statement.expression.kind==="raw_call"){
        const callee=context.symbols.get(statement.expression.name);
        if(callee?.closureKind==="mutable"){
          const expanded=emitMutableClosureCall(statement.expression,callee,context,statementPath,statement.name);
          context.symbols.set(statement.name,{kind:"local",type:"i64",id:expanded.resultLocal});
          emitted.push(...expanded.statements);continue;
        }
      }
      const type = controlExpressionType(statement.expression, context);
      const initializer = emitControlExpression(statement.expression, type, context, `${statementPath}.initializer`);
      const place = stableID("execution", context.description.id, statementPath, "place");
      const declaration = stableID("execution", context.description.id, statementPath, "declare");
      context.additions.push(graphEntity(place, entity(place, schema.mutablePlace, [[0x9e00, bytes(statement.name)], [0x9e01, ref(typeID(type))], [0x9e02, ref(initializer)]])));
      context.additions.push(graphEntity(declaration, entity(declaration, schema.declarePlace, [[0x9e10, ref(place)]])));
      const returnedClosure=statement.expression.kind==="raw_call"?context.descriptionsByName.get(statement.expression.name)?.expression:null;
      context.symbols.set(statement.name, { kind: "place", type, id: place, closureKind:statement.expression.kind==="mutable_closure"?"mutable":statement.expression.kind==="immutable_closure"?"immutable":returnedClosure?.kind==="canonical_closure"?(returnedClosure.mutable?"mutable":"immutable"):undefined, closureCapturePlace:statement.expression.kind==="mutable_closure"?context.symbols.get(statement.expression.capture)?.id:undefined }); emitted.push(declaration); continue;
    }
    if (statement.kind === "assign") {
      const symbol = context.symbols.get(statement.name); if (!symbol || symbol.kind !== "place") fail("lua.assignment_scope", statement.location);
      if(statement.expression.kind==="raw_call") {const callee=context.symbols.get(statement.expression.name);if(callee?.type==="function:i64:i64"){const expanded=emitMutableClosureCall(statement.expression,callee,context,statementPath,null,symbol.id);emitted.push(...expanded.statements);continue;}}
      const value = emitControlExpression(statement.expression, symbol.type, context, `${statementPath}.value`);
      const id = stableID("execution", context.description.id, statementPath, "assign"); context.additions.push(graphEntity(id, entity(id, schema.assignPlace, [[0x9e30, ref(symbol.id)], [0x9e31, ref(value)]]))); emitted.push(id); continue;
    }
    if (statement.kind === "return") {
      const value = emitControlExpression(statement.expression, context.description.resultType, context, `${statementPath}.value`);
      const id = stableID("execution", context.description.id, statementPath, "return"); context.additions.push(graphEntity(id, entity(id, schema.returned, [[0x9810, refs([value])]]))); emitted.push(id); terminal = true; continue;
    }
    if (statement.kind === "return_option_match_block") {
      const optionType = controlExpressionType(statement.value, context);
      if (!optionType.startsWith("option:")) fail("lua.option_match_value_type", statement.location);
      const payloadType = optionType.slice(7);
      const value = emitControlExpression(statement.value, optionType, context, `${statementPath}.value`);
      const noneValue = emitControlExpression(statement.none, context.description.resultType, context, `${statementPath}.none.value`);
      const noneBlock = returnedExpressionBlock(noneValue, context, `${statementPath}.none`);
      const binding = stableID("execution", context.description.id, statementPath, "some-binding");
      context.additions.push(graphEntity(binding, entity(binding, schema.variantBinding, [[0xa0600, bytes(statement.binding)], [0xa0601, ref(typeID(payloadType))]])));
      const childContext = { ...context, symbols: new Map(context.symbols) };
      childContext.symbols.set(statement.binding, { kind: "variant", type: payloadType, id: binding });
      const someBlock = emitControlBlock(statement.body, childContext, `${statementPath}.some`, true);
      const match = stableID("execution", context.description.id, statementPath, "option-match");
      context.additions.push(graphEntity(match, entity(match, schema.optionMatch, [[0xa0630, ref(value)], [0xa0631, ref(noneBlock)], [0xa0632, ref(binding)], [0xa0633, ref(someBlock)]])));
      const returned = stableID("execution", context.description.id, statementPath, "return");
      context.additions.push(graphEntity(returned, entity(returned, schema.returned, [[0x9810, refs([match])]])));
      emitted.push(returned); terminal = true; continue;
    }
    if(statement.kind==="return_result_match_block"){
      const resultType=controlExpressionType(statement.value,context);if(!resultType.startsWith("result:"))fail("lua.result_match_value_type",statement.location);const[okType,errorType]=splitResult(resultType),value=emitControlExpression(statement.value,resultType,context,`${statementPath}.value`),okBinding=stableID("execution",context.description.id,statementPath,"ok-binding"),errorBinding=stableID("execution",context.description.id,statementPath,"error-binding");context.additions.push(graphEntity(okBinding,entity(okBinding,schema.variantBinding,[[0xa0600,bytes(statement.okBinding)],[0xa0601,ref(typeID(okType))]])),graphEntity(errorBinding,entity(errorBinding,schema.variantBinding,[[0xa0600,bytes(statement.errorBinding)],[0xa0601,ref(typeID(errorType))]])));const okContext={...context,symbols:new Map(context.symbols)};okContext.symbols.set(statement.okBinding,{kind:"variant",type:okType,id:okBinding});const errorContext={...context,symbols:new Map(context.symbols)};errorContext.symbols.set(statement.errorBinding,{kind:"variant",type:errorType,id:errorBinding});const okBlock=emitControlBlock(statement.okBody,okContext,`${statementPath}.ok`,true),errorBlock=emitControlBlock(statement.errorBody,errorContext,`${statementPath}.error`,true),match=stableID("execution",context.description.id,statementPath,"result-match"),returned=stableID("execution",context.description.id,statementPath,"return");context.additions.push(graphEntity(match,entity(match,schema.resultMatch,[[0xa0620,ref(value)],[0xa0621,ref(okBinding)],[0xa0622,ref(okBlock)],[0xa0623,ref(errorBinding)],[0xa0624,ref(errorBlock)]])),graphEntity(returned,entity(returned,schema.returned,[[0x9810,refs([match])]])));emitted.push(returned);terminal=true;continue;
    }
    if(statement.kind==="effect"){const argument=emitControlExpression(statement.expression,"bool",context,`${statementPath}.argument`),capability=stableID("capability","observability.log"),effect=stableID("effect","observability.log"),id=stableID("execution",context.description.id,statementPath,"effect-invoke");if(!context.additions.some((item)=>item.id===capability))context.additions.push(graphEntity(capability,entity(capability,schema.capability,[[0x160,bytes("observability.log")]])),graphEntity(effect,entity(effect,schema.effect,[[0x150,bytes("observability.log")],[0x151,ref(capability)]])));context.additions.push(graphEntity(id,entity(id,schema.effectInvoke,[[0x9f10,ref(effect)],[0x9f11,refs([argument])]])));emitted.push(id);continue;}
    if(statement.kind==="call_statement"){
      const callee=context.symbols.get(statement.expression.name);
      if(!callee||callee.type!=="function:i64:i64")fail("lua.call_statement_profile",statement.location);
      const expanded=emitMutableClosureCall(statement.expression,callee,context,statementPath,null);
      emitted.push(...expanded.statements);continue;
    }
    if(statement.kind==="if_else"){
      const terminating=controlStatementsTerminate(statement.whenTrue)&&controlStatementsTerminate(statement.whenFalse),condition=emitControlExpression(statement.condition,"bool",context,`${statementPath}.condition`),trueBlock=emitControlBlock(statement.whenTrue,{...context,symbols:new Map(context.symbols)},`${statementPath}.true`,terminating),falseBlock=emitControlBlock(statement.whenFalse,{...context,symbols:new Map(context.symbols)},`${statementPath}.false`,terminating),id=stableID("execution",context.description.id,statementPath,"branch");context.additions.push(graphEntity(id,entity(id,schema.branch,[[0x9c00,ref(condition)],[0x9c01,ref(trueBlock)],[0x9c02,ref(falseBlock)]])));emitted.push(id);terminal=terminating;continue;
    }
    if (statement.kind === "while" || statement.kind === "when") {
      const condition = emitControlExpression(statement.condition, "bool", context, `${statementPath}.condition`);
      const childContext = { ...context, symbols: new Map(context.symbols) };
      const body = emitControlBlock(statement.body, childContext, `${statementPath}.body`, false);
      const id = stableID("execution", context.description.id, statementPath, statement.kind);
      const schemaID = statement.kind === "while" ? schema.whileStatement : schema.whenStatement, fields = statement.kind === "while" ? [[0x9e40, ref(condition)], [0x9e41, ref(body)]] : [[0x9f00, ref(condition)], [0x9f01, ref(body)]];
      context.additions.push(graphEntity(id, entity(id, schemaID, fields))); emitted.push(id); continue;
    }
  }
  if (requireReturn && !terminal) fail("lua.requires_return", context.description.location);
  const block = stableID("execution", context.description.id, path, "block"); context.additions.push(graphEntity(block, entity(block, schema.block, [[0x9800, refs(emitted)]]))); return block;
}
function controlStatementsTerminate(statements){const last=statements.at(-1);return!!last&&(["return","return_option_match_block","return_result_match_block"].includes(last.kind)||(last.kind==="if_else"&&controlStatementsTerminate(last.whenTrue)&&controlStatementsTerminate(last.whenFalse)));}

function emitProtocolDispatch(expression, context, path) {
  const whenFalse = context.implementations.get(expression.whenFalse), whenTrue = context.implementations.get(expression.whenTrue);
  if (!whenFalse || !whenTrue || whenFalse.protocol.id !== whenTrue.protocol.id) fail("lua.protocol_dispatch_implementation", expression.location);
  const requirement = whenFalse.protocol.requirements.find((item) => item.name === expression.requirement);
  const scalar = expression.kind === "protocol_dispatch_value", record = scalar ? null : context.records.get(expression.record);
  const fieldIndex = record?.fields.findIndex((item) => item.name === expression.field) ?? -1;
  if (!requirement || (scalar ? whenFalse.concreteID !== ids.i64 || whenTrue.concreteID !== ids.i64 : fieldIndex < 0 || record.fields.length !== 1 || record.fields[fieldIndex].type !== "i64" || whenFalse.concreteID !== record.id || whenTrue.concreteID !== record.id)) fail("lua.protocol_dispatch_contract", expression.location);
  const condition = parameterRead(expression.condition, "bool", context, `${path}.condition`, expression.location);
  const makeCall = (implementation, side) => {
    const value = parameterRead(expression.receiver, "i64", context, `${path}.${side}.receiver${scalar ? "" : ".field"}`, expression.location);
    const construct = scalar ? value : stableID("execution", context.description.id, path, side, "record");
    if (!scalar) context.additions.push(graphEntity(construct, entity(construct, schema.recordConstruct, [[0x9330, ref(record.id)], [0x9331, refs([value])]])));
    const boxed = stableID("execution", context.description.id, path, side, "interface-value");
    context.additions.push(graphEntity(boxed, entity(boxed, schema.interfaceValue, [[0xa0130, ref(implementation.protocol.id)], [0xa0131, ref(construct)], [0xa0132, ref(implementation.witnessID)]])));
    const argument = parameterRead(expression.argument, "i64", context, `${path}.${side}.argument`, expression.location);
    const call = stableID("execution", context.description.id, path, side, "dynamic-call");
    context.additions.push(graphEntity(call, entity(call, schema.dynamicMethodCall, [[0xa0140, ref(boxed)], [0xa0141, ref(requirement.id)], [0xa0142, refs([argument])]])));
    const returned = stableID("execution", context.description.id, path, side, "return"), block = stableID("execution", context.description.id, path, side, "block");
    context.additions.push(graphEntity(returned, entity(returned, schema.returned, [[0x9810, refs([call])]])), graphEntity(block, entity(block, schema.block, [[0x9800, refs([returned])]])));
    return block;
  };
  const falseBlock = makeCall(whenFalse, "false"), trueBlock = makeCall(whenTrue, "true");
  const branch = stableID("execution", context.description.id, path, "branch"), block = stableID("execution", context.description.id, path, "block");
  context.additions.push(graphEntity(branch, entity(branch, schema.branch, [[0x9c00, ref(condition)], [0x9c01, ref(trueBlock)], [0x9c02, ref(falseBlock)]])), graphEntity(block, entity(block, schema.block, [[0x9800, refs([branch])]])));
  return block;
}

function closureTypes(context) {
  const functionType = stableID("execution", "type", "function", "i64", "i64"), transitionType = stableID("execution", "type", "state-transition", functionType, ids.i64);
  if (!context.additions.some((item) => item.id === functionType)) context.additions.push(graphEntity(functionType, entity(functionType, schema.functionType, [[0xa0200, refs([ids.i64])], [0xa0201, ref(ids.i64)]])));
  if (!context.additions.some((item) => item.id === transitionType)) context.additions.push(graphEntity(transitionType, entity(transitionType, schema.transitionType, [[0xa0040, ref(functionType)], [0xa0041, ref(ids.i64)]])));
  return { functionType, transitionType };
}
function emitImmutableClosureRun(expression, context, path) {
  if (context.description.resultType !== "i64") fail("lua.closure_result_type", expression.location);
  const { functionType } = closureTypes(context), captureValue = parameterRead(expression.base, "i64", context, `${path}.closure.capture.value`, expression.location);
  const capture = stableID("execution", context.description.id, path, "capture"), parameter = stableID("execution", context.description.id, path, "closure-parameter"), captureRead = stableID("execution", context.description.id, path, "body.capture-read"), argumentRead = stableID("execution", context.description.id, path, "body.parameter-read"), add = stableID("execution", context.description.id, path, "body.add"), closure = stableID("execution", context.description.id, path, "closure"), callArgument = parameterRead(expression.value, "i64", context, `${path}.call.argument`, expression.location), call = stableID("execution", context.description.id, path, "indirect-call"), returned = stableID("execution", context.description.id, path, "return"), block = stableID("execution", context.description.id, path, "block");
  context.additions.push(graphEntity(capture,entity(capture,schema.captureBinding,[[0xa0210,bytes("base")],[0xa0211,ref(ids.i64)],[0xa0212,ref(captureValue)]])),graphEntity(parameter,entity(parameter,schema.parameter,[[0x9120,bytes("value")],[0x9121,ref(ids.i64)],[0x9122,"uu 0"]])),graphEntity(captureRead,entity(captureRead,schema.captureRead,[[0xa0220,ref(capture)]])),graphEntity(argumentRead,entity(argumentRead,schema.read,[[0x9130,ref(parameter)]])),graphEntity(add,entity(add,schema.integerAdd,[[0x9140,ref(captureRead)],[0x9141,ref(argumentRead)],[0x9142,ref(ids.i64)]])),graphEntity(closure,entity(closure,schema.closureConstruct,[[0xa0230,ref(functionType)],[0xa0231,refs([parameter])],[0xa0232,refs([capture])],[0xa0233,ref(add)]])),graphEntity(call,entity(call,schema.indirectCall,[[0xa0240,ref(closure)],[0xa0241,refs([callArgument])]])),graphEntity(returned,entity(returned,schema.returned,[[0x9810,refs([call])]])),graphEntity(block,entity(block,schema.block,[[0x9800,refs([returned])]]))); return block;
}
function emitMutableClosureRun(expression, context, path) {
  if (context.description.resultType !== "i64") fail("lua.closure_result_type", expression.location);
  const { functionType, transitionType }=closureTypes(context), initial=parameterRead(expression.start,"i64",context,`${path}.closure.capture.initial`,expression.location),capture=stableID("execution",context.description.id,path,"mutable-capture"),parameter=stableID("execution",context.description.id,path,"closure-parameter"),oldRead=stableID("execution",context.description.id,path,"closure.old"),deltaRead=stableID("execution",context.description.id,path,"closure.delta"),add=stableID("execution",context.description.id,path,"closure.add"),update=stableID("execution",context.description.id,path,"closure.update"),resultRead=stableID("execution",context.description.id,path,"closure.result"),sequence=stableID("execution",context.description.id,path,"closure.sequence"),closure=stableID("execution",context.description.id,path,"closure"),place=stableID("execution",context.description.id,path,"place"),declare=stableID("execution",context.description.id,path,"declare"),placeRead1=stableID("execution",context.description.id,path,"place-read-1"),firstArg=parameterRead(expression.first,"i64",context,`${path}.first.argument`,expression.location),firstCall=stableID("execution",context.description.id,path,"first-call"),firstLocal=stableID("execution",context.description.id,path,"first-local"),bindFirst=stableID("execution",context.description.id,path,"bind-first"),firstRead=stableID("execution",context.description.id,path,"first-read"),state=stableID("execution",context.description.id,path,"state"),assign=stableID("execution",context.description.id,path,"assign"),placeRead2=stableID("execution",context.description.id,path,"place-read-2"),secondArg=parameterRead(expression.second,"i64",context,`${path}.second.argument`,expression.location),secondCall=stableID("execution",context.description.id,path,"second-call"),result=stableID("execution",context.description.id,path,"result"),returned=stableID("execution",context.description.id,path,"return"),block=stableID("execution",context.description.id,path,"block");
  context.additions.push(graphEntity(capture,entity(capture,schema.mutableCaptureBinding,[[0xa0300,bytes("total")],[0xa0301,ref(ids.i64)],[0xa0302,ref(initial)]])),graphEntity(parameter,entity(parameter,schema.parameter,[[0x9120,bytes("delta")],[0x9121,ref(ids.i64)],[0x9122,"uu 0"]])),graphEntity(oldRead,entity(oldRead,schema.mutableCaptureRead,[[0xa0310,ref(capture)]])),graphEntity(deltaRead,entity(deltaRead,schema.read,[[0x9130,ref(parameter)]])),graphEntity(add,entity(add,schema.integerAdd,[[0x9140,ref(oldRead)],[0x9141,ref(deltaRead)],[0x9142,ref(ids.i64)]])),graphEntity(update,entity(update,schema.captureUpdate,[[0xa0320,ref(capture)],[0xa0321,ref(add)]])),graphEntity(resultRead,entity(resultRead,schema.mutableCaptureRead,[[0xa0310,ref(capture)]])),graphEntity(sequence,entity(sequence,schema.sequence,[[0xa0330,refs([update])],[0xa0331,ref(resultRead)]])),graphEntity(closure,entity(closure,schema.mutableClosureConstruct,[[0xa0340,ref(functionType)],[0xa0341,refs([parameter])],[0xa0342,refs([capture])],[0xa0343,ref(sequence)]])),graphEntity(place,entity(place,schema.mutablePlace,[[0x9e00,bytes("counter")],[0x9e01,ref(functionType)],[0x9e02,ref(closure)]])),graphEntity(declare,entity(declare,schema.declarePlace,[[0x9e10,ref(place)]])),graphEntity(placeRead1,entity(placeRead1,schema.placeRead,[[0x9e20,ref(place)]])),graphEntity(firstCall,entity(firstCall,schema.statefulIndirectCall,[[0xa0350,ref(placeRead1)],[0xa0351,refs([firstArg])]])),graphEntity(firstLocal,entity(firstLocal,schema.localBinding,[[0x9d00,bytes("first")],[0x9d01,ref(transitionType)],[0x9d02,ref(firstCall)]])),graphEntity(bindFirst,entity(bindFirst,schema.bindLocal,[[0x9d10,ref(firstLocal)]])),graphEntity(firstRead,entity(firstRead,schema.localRead,[[0x9d20,ref(firstLocal)]])),graphEntity(state,entity(state,schema.transitionState,[[0xa0060,ref(firstRead)]])),graphEntity(assign,entity(assign,schema.assignPlace,[[0x9e30,ref(place)],[0x9e31,ref(state)]])),graphEntity(placeRead2,entity(placeRead2,schema.placeRead,[[0x9e20,ref(place)]])),graphEntity(secondCall,entity(secondCall,schema.statefulIndirectCall,[[0xa0350,ref(placeRead2)],[0xa0351,refs([secondArg])]])),graphEntity(result,entity(result,schema.transitionResult,[[0xa0070,ref(secondCall)]])),graphEntity(returned,entity(returned,schema.returned,[[0x9810,refs([result])]])),graphEntity(block,entity(block,schema.block,[[0x9800,refs([declare,bindFirst,assign,returned])]]))); return block;
}
function emitTransitionStep(expression,context,path){const type=context.description.resultType;if(!type.startsWith("transition:"))fail("lua.transition_result_type",expression.location);const parts=type.split(":"),record=context.records.get(parts[2]),sourceType=`record:${parts[1]}:${parts[2]}`;if(!record||record.fields.length!==1||record.fields[0].name!==expression.field||record.fields[0].type!=="i64"||parts[3]!=="i64")fail("lua.transition_contract",expression.location);const recordRead=(suffix)=>{const base=parameterRead(expression.value,sourceType,context,`${path}.${suffix}.record`,expression.location),id=stableID("execution",context.description.id,path,suffix,"field-read");context.additions.push(graphEntity(id,entity(id,schema.fieldRead,[[0x9320,ref(base)],[0x9321,ref(stableID("execution",record.id,"field","0"))]])));return id;},oldState=recordRead("result"),left=recordRead("state.left"),right=parameterRead(expression.delta,"i64",context,`${path}.state.right`,expression.location),add=stableID("execution",context.description.id,path,"state-add"),construct=stableID("execution",context.description.id,path,"state-record"),transition=stableID("execution",context.description.id,path,"transition"),returned=stableID("execution",context.description.id,path,"return"),block=stableID("execution",context.description.id,path,"block");context.additions.push(graphEntity(add,entity(add,schema.integerAdd,[[0x9140,ref(left)],[0x9141,ref(right)],[0x9142,ref(ids.i64)]])),graphEntity(construct,entity(construct,schema.recordConstruct,[[0x9330,ref(record.id)],[0x9331,refs([add])]])),graphEntity(transition,entity(transition,schema.stateTransition,[[0xa0050,ref(typeID(type))],[0xa0051,ref(construct)],[0xa0052,ref(oldState)]])),graphEntity(returned,entity(returned,schema.returned,[[0x9810,refs([transition])]])),graphEntity(block,entity(block,schema.block,[[0x9800,refs([returned])]])));return block;}
function resultLiteral(value,context,path){const id=stableID("execution",context.description.id,path,"literal");context.additions.push(graphEntity(id,entity(id,schema.integerLiteral,[[0x9700,`uu ${value}`],[0x9701,ref(ids.i64)]])));return id;}
function resultBlock(value,context,path){const returned=stableID("execution",context.description.id,path,"return"),block=stableID("execution",context.description.id,path,"block");context.additions.push(graphEntity(returned,entity(returned,schema.returned,[[0x9810,refs([value])]])),graphEntity(block,entity(block,schema.block,[[0x9800,refs([returned])]])));return block;}
function emitCheckPositive(expression,context,path){if(context.description.resultType!=="result:i64:i64")fail("lua.result_profile",expression.location);const value1=parameterRead(expression.value,"i64",context,`${path}.condition.value`,expression.location),zero=resultLiteral("0",context,`${path}.condition.zero`),condition=stableID("execution",context.description.id,path,"condition"),resultType=typeID("result:i64:i64"),bad=resultLiteral("99",context,`${path}.error.payload`),errorValue=stableID("execution",context.description.id,path,"error"),goodRead=parameterRead(expression.value,"i64",context,`${path}.ok.payload`,expression.location),okValue=stableID("execution",context.description.id,path,"ok"),branch=stableID("execution",context.description.id,path,"branch"),block=stableID("execution",context.description.id,path,"block");context.additions.push(graphEntity(condition,entity(condition,schema.integerLessEqual,[[0x9160,ref(value1)],[0x9161,ref(zero)],[0x9162,ref(ids.i64)]])),graphEntity(errorValue,entity(errorValue,schema.resultError,[[0x9420,ref(resultType)],[0x9421,ref(bad)]])),graphEntity(okValue,entity(okValue,schema.resultOk,[[0x9410,ref(resultType)],[0x9411,ref(goodRead)]])));const errorBlock=resultBlock(errorValue,context,`${path}.then`),okBlock=resultBlock(okValue,context,`${path}.else`);context.additions.push(graphEntity(branch,entity(branch,schema.branch,[[0x9c00,ref(condition)],[0x9c01,ref(errorBlock)],[0x9c02,ref(okBlock)]])),graphEntity(block,entity(block,schema.block,[[0x9800,refs([branch])]])));return block;}
function emitIncrementPositive(expression,context,path){if(context.description.resultType!=="result:i64:i64")fail("lua.result_profile",expression.location);const callee=context.descriptionsByName.get("CheckPositive");if(!callee)fail("lua.result_check_scope",expression.location);const argument=parameterRead(expression.value,"i64",context,`${path}.call.argument`,expression.location),call=stableID("execution",context.description.id,path,"call"),okBinding=stableID("execution",context.description.id,path,"ok-binding"),errorBinding=stableID("execution",context.description.id,path,"error-binding"),okRead=stableID("execution",context.description.id,path,"ok-read"),one=resultLiteral("1",context,`${path}.one`),add=stableID("execution",context.description.id,path,"add"),okValue=stableID("execution",context.description.id,path,"ok"),errorRead=stableID("execution",context.description.id,path,"error-read"),errorValue=stableID("execution",context.description.id,path,"error"),match=stableID("execution",context.description.id,path,"match"),returned=stableID("execution",context.description.id,path,"return"),block=stableID("execution",context.description.id,path,"block"),resultType=typeID("result:i64:i64");context.additions.push(graphEntity(call,entity(call,schema.call,[[0x9600,ref(callee.id)],[0x9601,refs([argument])]])),graphEntity(okBinding,entity(okBinding,schema.variantBinding,[[0xa0600,bytes("accepted")],[0xa0601,ref(ids.i64)]])),graphEntity(errorBinding,entity(errorBinding,schema.variantBinding,[[0xa0600,bytes("error")],[0xa0601,ref(ids.i64)]])),graphEntity(okRead,entity(okRead,schema.variantRead,[[0xa0610,ref(okBinding)]])),graphEntity(add,entity(add,schema.integerAdd,[[0x9140,ref(okRead)],[0x9141,ref(one)],[0x9142,ref(ids.i64)]])),graphEntity(okValue,entity(okValue,schema.resultOk,[[0x9410,ref(resultType)],[0x9411,ref(add)]])),graphEntity(errorRead,entity(errorRead,schema.variantRead,[[0xa0610,ref(errorBinding)]])),graphEntity(errorValue,entity(errorValue,schema.resultError,[[0x9420,ref(resultType)],[0x9421,ref(errorRead)]])));const okBlock=resultBlock(okValue,context,`${path}.ok-arm`),errorBlock=resultBlock(errorValue,context,`${path}.error-arm`);context.additions.push(graphEntity(match,entity(match,schema.resultMatch,[[0xa0620,ref(call)],[0xa0621,ref(okBinding)],[0xa0622,ref(okBlock)],[0xa0623,ref(errorBinding)],[0xa0624,ref(errorBlock)]])),graphEntity(returned,entity(returned,schema.returned,[[0x9810,refs([match])]])),graphEntity(block,entity(block,schema.block,[[0x9800,refs([returned])]])));return block;}

function controlExpressionType(expression, context) {
  if(expression.kind==="ok"||expression.kind==="err")return context.description.resultType;
  if(expression.kind==="record_construct"){const record=context.records.get(expression.name);if(!record)fail("lua.record_construct_type",expression.location);return `record:${record.id}:${record.name}`;}
  if(expression.kind==="transition_construct"){const state=controlExpressionType(expression.arguments[0],context),result=controlExpressionType(expression.arguments[1],context);if(!state.startsWith("record:"))fail("lua.transition_construct_type",expression.location);const parts=state.split(":");return `transition:${parts[1]}:${parts.slice(2).join(":")}:${result}`;}
  if (expression.kind === "integer_literal" || expression.kind === "add" || expression.kind === "multiply"||expression.kind==="subtract") return "i64";
  if(expression.kind==="bool_literal")return "bool";
  if(expression.kind==="mutable_closure"||expression.kind==="immutable_closure")return "function:i64:i64";
  if(expression.kind==="canonical_closure")return"function:i64:i64";
  if(expression.kind==="raw_call"){
    const local=context.symbols.get(expression.name);
    if(local?.type==="function:i64:i64")return "i64";
    const declared=context.descriptionsByName.get(expression.name);if(!declared)fail("lua.unknown_call",expression.location);return declared.resultType;
  }
  if(expression.kind==="interface_value") {const implementation=context.implementations.get(expression.implementation);if(!implementation)fail("lua.interface_implementation",expression.location);return`interface:${implementation.protocol.id}:${implementation.protocol.name}`;}
  if(expression.kind==="protocol_call"){const receiver=controlExpressionType(expression.receiver,context);if(!receiver.startsWith("interface:"))fail("lua.protocol_call_receiver",expression.location);return"i64";}
  if(expression.kind==="map_lookup")return inferComposableExpression(expression,context);
  if(expression.kind==="slice")return "slice:i64";
  if(expression.kind==="length"||expression.kind==="index")return "i64";
  if(expression.kind==="collection_append"||expression.kind==="collection_update"||expression.kind==="slice_remove"){const base=context.symbols.get(expression.base);if(!base||base.type!=="slice:i64")fail("lua.collection_argument_type",expression.location);return base.type;}
  if(expression.kind==="map_update"||expression.kind==="map_remove"){const base=context.symbols.get(expression.base);if(!base||!base.type.startsWith("map:"))fail("lua.map_argument_type",expression.location);return base.type;}
  if(expression.kind.startsWith("nested_")){
    const kind=expression.kind.slice(7);if(["collection_append","collection_update","slice_remove","slice"].includes(kind))return "slice:i64";if(["length","index_zero","fold","lookup_zero"].includes(kind))return "i64";if(kind==="map_update"||kind==="map_remove")return controlExpressionType(expression.arguments[0],context);
  }
  if (expression.kind === "index") { const symbol=context.symbols.get(expression.base); if(!symbol||symbol.type!=="slice:i64")fail("lua.dynamic_index_collection_type",expression.location); const index=context.symbols.get(expression.index); if(!index||index.type!=="i64")fail("lua.dynamic_index_type",expression.location); return "i64"; }
  if (["less_equal", "equal_i64", "boolean_and", "boolean_or"].includes(expression.kind)) return "bool";
  if (expression.kind === "boolean_boundary") { const symbol=context.symbols.get(expression.name); if(!symbol||symbol.type!=="bool")fail("lua.boolean_boundary_type",expression.location); return "bool"; }
  if (expression.kind === "identifier") { const symbol = context.symbols.get(expression.name); if (!symbol) fail("lua.unknown_identifier", expression.location); return symbol.type; }
  if(expression.kind==="field") { const symbol=context.symbols.get(expression.base);if(!symbol||!symbol.type.startsWith("record:"))fail("lua.field_receiver_type",expression.location);const record=context.records.get(symbol.type.split(":").slice(2).join(":")),member=record?.fields.find(item=>item.name===expression.field);if(!member)fail("lua.unknown_field",expression.location);return member.type; }
  fail("lua.control_expression_profile", expression.location);
}
function emitControlExpression(expression, expected, context, path) {
  if(expression.kind==="mutable_closure"||expression.kind==="immutable_closure")return emitRawClosure(expression,expected,context,path);
  if(expression.kind==="canonical_closure")return emitCanonicalClosure(expression,expected,context,path);
  if(expression.kind==="map_lookup")return emitComposableExpression(expression,context,path).id;
  if(expression.kind.startsWith("nested_"))return emitNestedControlExpression(expression,expected,context,path);
  if(expression.kind==="ok"||expression.kind==="err"){
    if(!expected.startsWith("result:")||expression.arguments.length!==1)fail("lua.result_constructor_type",expression.location);const[okType,errorType]=splitResult(expected),payloadType=expression.kind==="ok"?okType:errorType,payload=emitControlExpression(expression.arguments[0],payloadType,context,`${path}.payload`),id=stableID("execution",context.description.id,path,expression.kind),shape=expression.kind==="ok"?[schema.resultOk,0x9410,0x9411]:[schema.resultError,0x9420,0x9421];ensureType(expected,context.additions,context.records);context.additions.push(graphEntity(id,entity(id,shape[0],[[shape[1],ref(typeID(expected))],[shape[2],ref(payload)]])));return id;
  }
  if(expression.kind==="record_construct"){
    if(!expected.startsWith("record:"))fail("lua.record_construct_type",expression.location);const record=context.records.get(expected.split(":").slice(2).join(":"));if(!record||record.name!==expression.name||expression.fields.length!==record.fields.length||expression.fields.some((item,index)=>item.name!==record.fields[index].name))fail("lua.record_construct_fields",expression.location);const values=expression.fields.map((item,index)=>emitControlExpression(item.value,record.fields[index].type,context,`${path}.field.${index}`)),id=stableID("execution",context.description.id,path,"record-construct");context.additions.push(graphEntity(id,entity(id,schema.recordConstruct,[[0x9330,ref(record.id)],[0x9331,refs(values)]])));return id;
  }
  if(expression.kind==="transition_construct"){
    if(!expected.startsWith("transition:")||expression.arguments.length!==2)fail("lua.transition_construct_type",expression.location);const parts=expected.split(":"),stateType=`record:${parts[1]}:${parts[2]}`,resultType=parts[3],state=emitControlExpression(expression.arguments[0],stateType,context,`${path}.state`),result=emitControlExpression(expression.arguments[1],resultType,context,`${path}.result`),id=stableID("execution",context.description.id,path,"transition");ensureType(expected,context.additions,context.records);context.additions.push(graphEntity(id,entity(id,schema.stateTransition,[[0xa0050,ref(typeID(expected))],[0xa0051,ref(state)],[0xa0052,ref(result)]])));return id;
  }
  const actual = controlExpressionType(expression, context); if (actual !== expected) fail("lua.control_expression_type", expression.location);
  if (expression.kind === "identifier") { const symbol = context.symbols.get(expression.name); const id = stableID("execution", context.description.id, path, "read"); const shape = symbol.kind === "parameter" ? [schema.read, 0x9130] : symbol.kind === "variant" ? [schema.variantRead, 0xa0610] : symbol.kind === "local" ? [schema.localRead,0x9d20] : [schema.placeRead, 0x9e20]; context.additions.push(graphEntity(id, entity(id, shape[0], [[shape[1], ref(symbol.id)]]))); return id; }
  if(expression.kind==="field"){const symbol=context.symbols.get(expression.base),base=emitControlExpression({kind:"identifier",name:expression.base,location:expression.location},symbol.type,context,`${path}.base`),record=context.records.get(symbol.type.split(":").slice(2).join(":")),index=record.fields.findIndex(item=>item.name===expression.field),id=stableID("execution",context.description.id,path,"field-read");context.additions.push(graphEntity(id,entity(id,schema.fieldRead,[[0x9320,ref(base)],[0x9321,ref(stableID("execution",record.id,"field",String(index)))]])));return id;}
  if(expression.kind==="length"){const base=emitControlExpression({kind:"identifier",name:expression.base,location:expression.location},"slice:i64",context,`${path}.collection`),id=stableID("execution",context.description.id,path,"length");context.additions.push(graphEntity(id,entity(id,schema.collectionLength,[[0x9f90,ref(base)]])));return id;}
  if (expression.kind === "boolean_boundary") return emitControlExpression({kind:"identifier",name:expression.name,location:expression.location}, expected, context, path);
  if(expression.kind==="bool_literal"){const id=stableID("execution",context.description.id,path,"bool-literal");context.additions.push(graphEntity(id,entity(id,schema.boolLiteral,[[0x9b00,expression.value?"tr":"fa"]])));return id;}
  if(expression.kind==="slice"){
    if(expected!=="slice:i64")fail("lua.slice_construct_type",expression.location);
    const values=expression.arguments.map((name,index)=>emitControlExpression({kind:"identifier",name,location:expression.location},"i64",context,`${path}.item.${index}`)),id=stableID("execution",context.description.id,path,"slice");
    context.additions.push(graphEntity(id,entity(id,schema.sliceConstruct,[[0xa0680,ref(typeID(expected))],[0xa0681,refs(values)]])));return id;
  }
  if (expression.kind === "integer_literal") { const id = stableID("execution", context.description.id, path, "literal"); context.additions.push(graphEntity(id, entity(id, schema.integerLiteral, [[0x9700, `uu ${i64Word(expression.value,expression.location)}`], [0x9701, ref(ids.i64)]]))); return id; }
  if(expression.kind==="index"){const base=emitControlExpression({kind:"identifier",name:expression.base,location:expression.location},"slice:i64",context,`${path}.collection`),index=emitControlExpression({kind:"identifier",name:expression.index,location:expression.location},"i64",context,`${path}.index`),id=stableID("execution",context.description.id,path,"dynamic-index");context.additions.push(graphEntity(id,entity(id,schema.dynamicIndexRead,[[0x9fa0,ref(base)],[0x9fa1,ref(index)]])));return id;}
  if(expression.kind==="collection_append"||expression.kind==="collection_update"){
    const base=emitControlExpression({kind:"identifier",name:expression.base,location:expression.location},expected,context,`${path}.collection`),value=emitControlExpression({kind:"identifier",name:expression.value,location:expression.location},"i64",context,`${path}.value`),id=stableID("execution",context.description.id,path,expression.kind);if(expression.kind==="collection_append")context.additions.push(graphEntity(id,entity(id,schema.collectionAppend,[[0x9fb0,ref(base)],[0x9fb1,ref(value)]])));else{const index=emitControlExpression({kind:"identifier",name:expression.index,location:expression.location},"i64",context,`${path}.index`);context.additions.push(graphEntity(id,entity(id,schema.collectionUpdate,[[0x9fc0,ref(base)],[0x9fc1,ref(index)],[0x9fc2,ref(value)]])));}return id;
  }
  if(expression.kind==="map_update"){
    const parts=expected.split(":"),base=emitControlExpression({kind:"identifier",name:expression.base,location:expression.location},expected,context,`${path}.map`),key=emitControlExpression({kind:"identifier",name:expression.key,location:expression.location},parts[1],context,`${path}.key`),value=emitControlExpression({kind:"identifier",name:expression.value,location:expression.location},parts[2],context,`${path}.value`),id=stableID("execution",context.description.id,path,"map-update");context.additions.push(graphEntity(id,entity(id,schema.mapUpdate,[[0xa0430,ref(base)],[0xa0431,ref(key)],[0xa0432,ref(value)]])));return id;
  }
  if(expression.kind==="raw_call"){
    const local=context.symbols.get(expression.name);
    if(local?.closureKind==="immutable"){
      if(expression.arguments.length!==1||expected!=="i64")fail("lua.indirect_call_type",expression.location);
      const callee=emitControlExpression({kind:"identifier",name:expression.name,location:expression.location},local.type,context,`${path}.callee`),argument=emitControlExpression(expression.arguments[0],"i64",context,`${path}.argument`),id=stableID("execution",context.description.id,path,"indirect-call");context.additions.push(graphEntity(id,entity(id,schema.indirectCall,[[0xa0240,ref(callee)],[0xa0241,refs([argument])]])));return id;
    }
    const declared=context.descriptionsByName.get(expression.name);if(!declared||declared.resultType!==expected||declared.parameters.length!==expression.arguments.length)fail("lua.call_type",expression.location);
    const arguments_=expression.arguments.map((item,index)=>emitControlExpression(item,declared.parameters[index].type,context,`${path}.argument.${index}`)),id=stableID("execution",context.description.id,path,"call");context.additions.push(graphEntity(id,entity(id,schema.call,[[0x9600,ref(declared.id)],[0x9601,refs(arguments_)]])));return id;
  }
  if(expression.kind==="interface_value"){
    const implementation=context.implementations.get(expression.implementation);if(!implementation||typeID(expected)!==implementation.protocol.id)fail("lua.interface_implementation",expression.location);const record=[...context.records.values()].find(item=>item.id===implementation.concreteID),expectedConcrete=implementation.concreteID===ids.i64?"i64":record?`record:${record.id}:${record.name}`:null;if(!expectedConcrete)fail("lua.interface_concrete_type",expression.location);const concrete=emitControlExpression(expression.value,expectedConcrete,context,`${path}.value`),id=stableID("execution",context.description.id,path,"interface-value");context.additions.push(graphEntity(id,entity(id,schema.interfaceValue,[[0xa0130,ref(implementation.protocol.id)],[0xa0131,ref(concrete)],[0xa0132,ref(implementation.witnessID)]])));return id;
  }
  if(expression.kind==="protocol_call"){
    const receiverType=controlExpressionType(expression.receiver,context),protocol=[...context.implementations.values()].map(item=>item.protocol).find(item=>item.id===typeID(receiverType)),requirement=protocol?.requirements.find(item=>item.name===expression.requirement);if(!requirement||expression.arguments.length!==1||expected!=="i64")fail("lua.protocol_call_contract",expression.location);const receiver=emitControlExpression(expression.receiver,receiverType,context,`${path}.receiver`),argument=emitControlExpression(expression.arguments[0],"i64",context,`${path}.argument`),id=stableID("execution",context.description.id,path,"dynamic-call");context.additions.push(graphEntity(id,entity(id,schema.dynamicMethodCall,[[0xa0140,ref(receiver)],[0xa0141,ref(requirement.id)],[0xa0142,refs([argument])]])));return id;
  }
  const leftType = expression.kind === "boolean_and" || expression.kind === "boolean_or" ? "bool" : "i64";
  const left = emitControlExpression(expression.left, leftType, context, `${path}.left`), right = emitControlExpression(expression.right, leftType, context, `${path}.right`), id = stableID("execution", context.description.id, path, expression.kind);
  if (expression.kind === "equal_i64") {
    const forward=stableID("execution",context.description.id,path,"less-equal-forward"),reverse=stableID("execution",context.description.id,path,"less-equal-reverse");
    context.additions.push(graphEntity(forward,entity(forward,schema.integerLessEqual,[[0x9160,ref(left)],[0x9161,ref(right)],[0x9162,ref(ids.i64)]])));context.additions.push(graphEntity(reverse,entity(reverse,schema.integerLessEqual,[[0x9160,ref(right)],[0x9161,ref(left)],[0x9162,ref(ids.i64)]])));context.additions.push(graphEntity(id,entity(id,schema.booleanAnd,[[0x9b10,ref(forward)],[0x9b11,ref(reverse)]])));return id;
  }
  const shape = expression.kind === "add" ? [schema.integerAdd,0x9140,0x9141,[[0x9142,ref(ids.i64)]]] : expression.kind === "multiply" ? [schema.integerMultiply,0x9900,0x9901,[[0x9902,ref(ids.i64)]]] : expression.kind==="subtract"?[schema.integerSubtract,0x9a00,0x9a01,[[0x9a02,ref(ids.i64)]]]:expression.kind === "boolean_and" ? [schema.booleanAnd,0x9b10,0x9b11,[]] : expression.kind === "boolean_or" ? [schema.booleanOr,0x9c10,0x9c11,[]] : [schema.integerLessEqual,0x9160,0x9161,[[0x9162,ref(ids.i64)]]];
  context.additions.push(graphEntity(id, entity(id, shape[0], [[shape[1],ref(left)],[shape[2],ref(right)],...shape[3]]))); return id;
}

function emitRawClosure(expression,expected,context,path){
  if(expected!=="function:i64:i64")fail("lua.closure_type",expression.location);
  ensureType(expected,context.additions,context.records);
  const captureSymbol=context.symbols.get(expression.capture??findClosureCapture(expression.body,expression.parameter,context));
  if(!captureSymbol||captureSymbol.type!=="i64")fail("lua.closure_capture_scope",expression.location);
  const captureName=expression.capture??findClosureCapture(expression.body,expression.parameter,context),parameter=stableID("execution",context.description.id,path,"parameter"),capture=stableID("execution",context.description.id,path,"capture"),initial=emitControlExpression({kind:"identifier",name:captureName,location:expression.location},"i64",context,`${path}.capture.initial`);
  context.additions.push(graphEntity(parameter,entity(parameter,schema.parameter,[[0x9120,bytes(expression.parameter)],[0x9121,ref(ids.i64)],[0x9122,"uu 0"]])));
  if(expression.kind==="mutable_closure"){
    const old=stableID("execution",context.description.id,path,"capture-read"),argument=stableID("execution",context.description.id,path,"parameter-read"),body=stableID("execution",context.description.id,path,"body"),update=stableID("execution",context.description.id,path,"update"),result=stableID("execution",context.description.id,path,"result"),sequence=stableID("execution",context.description.id,path,"sequence"),id=stableID("execution",context.description.id,path,"mutable-closure");
    context.additions.push(graphEntity(capture,entity(capture,schema.mutableCaptureBinding,[[0xa0300,bytes(captureName)],[0xa0301,ref(ids.i64)],[0xa0302,ref(initial)]])),graphEntity(old,entity(old,schema.mutableCaptureRead,[[0xa0310,ref(capture)]])),graphEntity(argument,entity(argument,schema.read,[[0x9130,ref(parameter)]])));
    emitClosureI64Body(expression.body,captureName,expression.parameter,old,argument,body,context);
    context.additions.push(graphEntity(update,entity(update,schema.captureUpdate,[[0xa0320,ref(capture)],[0xa0321,ref(body)]])),graphEntity(result,entity(result,schema.mutableCaptureRead,[[0xa0310,ref(capture)]])),graphEntity(sequence,entity(sequence,schema.sequence,[[0xa0330,refs([update])],[0xa0331,ref(result)]])),graphEntity(id,entity(id,schema.mutableClosureConstruct,[[0xa0340,ref(typeID(expected))],[0xa0341,refs([parameter])],[0xa0342,refs([capture])],[0xa0343,ref(sequence)]])));return id;
  }
  const captureRead=stableID("execution",context.description.id,path,"capture-read"),argumentRead=stableID("execution",context.description.id,path,"parameter-read"),body=stableID("execution",context.description.id,path,"body"),id=stableID("execution",context.description.id,path,"closure");
  context.additions.push(graphEntity(capture,entity(capture,schema.captureBinding,[[0xa0210,bytes(captureName)],[0xa0211,ref(ids.i64)],[0xa0212,ref(initial)]])),graphEntity(captureRead,entity(captureRead,schema.captureRead,[[0xa0220,ref(capture)]])),graphEntity(argumentRead,entity(argumentRead,schema.read,[[0x9130,ref(parameter)]])));
  emitClosureI64Body(expression.body,captureName,expression.parameter,captureRead,argumentRead,body,context);
  context.additions.push(graphEntity(id,entity(id,schema.closureConstruct,[[0xa0230,ref(typeID(expected))],[0xa0231,refs([parameter])],[0xa0232,refs([capture])],[0xa0233,ref(body)]])));return id;
}
function emitCanonicalClosure(expression,expected,context,path){
  if(expected!=="function:i64:i64")fail("lua.closure_type",expression.location);ensureType(expected,context.additions,context.records);const initial=emitControlExpression(expression.initial,"i64",context,`${path}.capture.initial`),capture=stableID("execution",context.description.id,path,"capture"),parameter=stableID("execution",context.description.id,path,"parameter"),captureRead=stableID("execution",context.description.id,path,"capture-read"),parameterReadID=stableID("execution",context.description.id,path,"parameter-read"),body=stableID("execution",context.description.id,path,"body"),id=stableID("execution",context.description.id,path,expression.mutable?"mutable-closure":"closure");context.additions.push(graphEntity(parameter,entity(parameter,schema.parameter,[[0x9120,bytes(expression.parameter)],[0x9121,ref(ids.i64)],[0x9122,"uu 0"]])));if(expression.mutable)context.additions.push(graphEntity(capture,entity(capture,schema.mutableCaptureBinding,[[0xa0300,bytes(expression.capture)],[0xa0301,ref(ids.i64)],[0xa0302,ref(initial)]])),graphEntity(captureRead,entity(captureRead,schema.mutableCaptureRead,[[0xa0310,ref(capture)]])));else context.additions.push(graphEntity(capture,entity(capture,schema.captureBinding,[[0xa0210,bytes(expression.capture)],[0xa0211,ref(ids.i64)],[0xa0212,ref(initial)]])),graphEntity(captureRead,entity(captureRead,schema.captureRead,[[0xa0220,ref(capture)]])));context.additions.push(graphEntity(parameterReadID,entity(parameterReadID,schema.read,[[0x9130,ref(parameter)]])));emitClosureI64Body(expression.body,expression.capture,expression.parameter,captureRead,parameterReadID,body,context);if(!expression.mutable){context.additions.push(graphEntity(id,entity(id,schema.closureConstruct,[[0xa0230,ref(typeID(expected))],[0xa0231,refs([parameter])],[0xa0232,refs([capture])],[0xa0233,ref(body)]])));return id;}const update=stableID("execution",context.description.id,path,"update"),result=stableID("execution",context.description.id,path,"result"),sequence=stableID("execution",context.description.id,path,"sequence");context.additions.push(graphEntity(update,entity(update,schema.captureUpdate,[[0xa0320,ref(capture)],[0xa0321,ref(body)]])),graphEntity(result,entity(result,schema.mutableCaptureRead,[[0xa0310,ref(capture)]])),graphEntity(sequence,entity(sequence,schema.sequence,[[0xa0330,refs([update])],[0xa0331,ref(result)]])),graphEntity(id,entity(id,schema.mutableClosureConstruct,[[0xa0340,ref(typeID(expected))],[0xa0341,refs([parameter])],[0xa0342,refs([capture])],[0xa0343,ref(sequence)]])));return id;
}
function emitNestedControlExpression(expression,expected,context,path){
  const kind=expression.kind.slice(7),args=expression.arguments,id=stableID("execution",context.description.id,path,kind),emit=(value,type,suffix)=>emitControlExpression(value,type,context,`${path}.${suffix}`);
  if(kind==="map_update"){
    if(!expected.startsWith("map:")||args.length!==3)fail("lua.map_mutation_type",expression.location);const parts=expected.split(":"),base=emit(args[0],expected,"map"),key=emit(args[1],parts[1],"key"),value=emit(args[2],parts[2],"value");context.additions.push(graphEntity(id,entity(id,schema.mapUpdate,[[0xa0430,ref(base)],[0xa0431,ref(key)],[0xa0432,ref(value)]])));return id;
  }
  if(kind==="collection_update"||kind==="collection_append"){
    if(expected!=="slice:i64")fail("lua.collection_argument_type",expression.location);const base=emit(args[0],expected,"collection"),value=emit(args.at(-1),"i64","value");if(kind==="collection_append")context.additions.push(graphEntity(id,entity(id,schema.collectionAppend,[[0x9fb0,ref(base)],[0x9fb1,ref(value)]])));else{const index=emit(args[1],"i64","index");context.additions.push(graphEntity(id,entity(id,schema.collectionUpdate,[[0x9fc0,ref(base)],[0x9fc1,ref(index)],[0x9fc2,ref(value)]])));}return id;
  }
  if(kind==="length"||kind==="index_zero"){
    if(expected!=="i64")fail("lua.collection_query_type",expression.location);const base=emit(args[0],"slice:i64","collection");if(kind==="length")context.additions.push(graphEntity(id,entity(id,schema.collectionLength,[[0x9f90,ref(base)]])));else{const index=emit(args[1],"i64","index");context.additions.push(graphEntity(id,entity(id,schema.dynamicIndexRead,[[0x9fa0,ref(base)],[0x9fa1,ref(index)]])));}return id;
  }
  fail("lua.nested_expression_profile",expression.location);
}
function findClosureCapture(expression,parameter,context){const names=[];const walk=value=>{if(!value||typeof value!=="object")return;if(value.kind==="identifier"&&value.name!==parameter)names.push(value.name);for(const child of Object.values(value))walk(child);};walk(expression);const unique=[...new Set(names.filter(name=>context.symbols.has(name)))];if(unique.length!==1)fail("lua.closure_capture_profile",expression.location);return unique[0];}
function emitClosureI64Body(expression,captureName,parameterName,captureRead,parameterReadID,id,context){if(expression.kind!=="add"&&expression.kind!=="multiply")fail("lua.closure_body_profile",expression.location);const read=value=>{if(value.kind!=="identifier")fail("lua.closure_operand",value.location);if(value.name===captureName)return captureRead;if(value.name===parameterName)return parameterReadID;fail("lua.closure_operand",value.location);};const shape=expression.kind==="add"?[schema.integerAdd,0x9140,0x9141,0x9142]:[schema.integerMultiply,0x9900,0x9901,0x9902];context.additions.push(graphEntity(id,entity(id,shape[0],[[shape[1],ref(read(expression.left))],[shape[2],ref(read(expression.right))],[shape[3],ref(ids.i64)]])));}
function emitMutableClosureCall(expression,callee,context,path,resultName,resultPlace=null){
  if(expression.arguments.length!==1)fail("lua.stateful_call_arity",expression.location);const calleeRead=emitControlExpression({kind:"identifier",name:expression.name,location:expression.location},callee.type,context,`${path}.callee`),argument=emitControlExpression(expression.arguments[0],"i64",context,`${path}.argument`),call=stableID("execution",context.description.id,path,"stateful-call"),transitionTypeID=stableID("execution","type","state-transition",typeID(callee.type),ids.i64);if(!context.additions.some(item=>item.id===transitionTypeID))context.additions.push(graphEntity(transitionTypeID,entity(transitionTypeID,schema.transitionType,[[0xa0040,ref(typeID(callee.type))],[0xa0041,ref(ids.i64)]])));context.additions.push(graphEntity(call,entity(call,schema.statefulIndirectCall,[[0xa0350,ref(calleeRead)],[0xa0351,refs([argument])]])));
  const transitionLocal=stableID("execution",context.description.id,path,"transition-local"),bindTransition=stableID("execution",context.description.id,path,"bind-transition"),transitionRead1=stableID("execution",context.description.id,path,"transition-read-state"),state=stableID("execution",context.description.id,path,"state"),assign=stableID("execution",context.description.id,path,"assign");context.additions.push(graphEntity(transitionLocal,entity(transitionLocal,schema.localBinding,[[0x9d00,bytes(`${expression.name}_transition`)],[0x9d01,ref(transitionTypeID)],[0x9d02,ref(call)]])),graphEntity(bindTransition,entity(bindTransition,schema.bindLocal,[[0x9d10,ref(transitionLocal)]])),graphEntity(transitionRead1,entity(transitionRead1,schema.localRead,[[0x9d20,ref(transitionLocal)]])),graphEntity(state,entity(state,schema.transitionState,[[0xa0060,ref(transitionRead1)]])),graphEntity(assign,entity(assign,schema.assignPlace,[[0x9e30,ref(callee.id)],[0x9e31,ref(state)]])));
  const statements=[bindTransition,assign];
  const makeResult=(suffix)=>{const read=stableID("execution",context.description.id,path,`transition-read-${suffix}`),result=stableID("execution",context.description.id,path,`result-${suffix}`);context.additions.push(graphEntity(read,entity(read,schema.localRead,[[0x9d20,ref(transitionLocal)]])),graphEntity(result,entity(result,schema.transitionResult,[[0xa0070,ref(read)]])));return result;};
  if(callee.closureCapturePlace){const captureResult=makeResult("capture"),captureAssign=stableID("execution",context.description.id,path,"assign-capture");context.additions.push(graphEntity(captureAssign,entity(captureAssign,schema.assignPlace,[[0x9e30,ref(callee.closureCapturePlace)],[0x9e31,ref(captureResult)]])));statements.push(captureAssign);}
  if(resultPlace){const placeResult=makeResult("place"),placeAssign=stableID("execution",context.description.id,path,"assign-result");context.additions.push(graphEntity(placeAssign,entity(placeAssign,schema.assignPlace,[[0x9e30,ref(resultPlace)],[0x9e31,ref(placeResult)]])));statements.push(placeAssign);}
  let resultLocal=null;if(resultName){const result=makeResult("value"),local=stableID("execution",context.description.id,path,"result-local"),bind=stableID("execution",context.description.id,path,"bind-result");context.additions.push(graphEntity(local,entity(local,schema.localBinding,[[0x9d00,bytes(resultName)],[0x9d01,ref(ids.i64)],[0x9d02,ref(result)]])),graphEntity(bind,entity(bind,schema.bindLocal,[[0x9d10,ref(local)]])));statements.push(bind);resultLocal=local;}return{statements,resultLocal};
}

function parseSource(source, file, records,protocolTypes=new Map()) {
  if (typeof source !== "string" || typeof file !== "string" || !file) fail("lua.invalid_source");
  const lines = source.replace(/\r\n?/g, "\n").split("\n");
  const declarations = [];
  let annotations = [];
  for (let index = 0; index < lines.length;) {
    const trimmed = lines[index].trim();
    if (!trimmed || trimmed.startsWith("--") && !trimmed.startsWith("---@")) { index += 1; continue; }
    if (trimmed.startsWith("---@class ")) { index += 1; while (index < lines.length && lines[index].trim().startsWith("---@field ")) index += 1; continue; }
    if (trimmed.startsWith("---@")) { annotations.push({ text: trimmed, line: index + 1 }); index += 1; continue; }
    const match = /^(local\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*$/.exec(trimmed);
    if (!match) fail("lua.unsupported_top_level", { file, line: index + 1, column: 1 });
    const location = { file, line: index + 1, column: lines[index].indexOf("function") + 1 };
    const parameters = match[3].trim() ? match[3].split(",").map((item) => item.trim()) : [];
    if (parameters.some((item) => !/^[A-Za-z_][A-Za-z0-9_]*$/.test(item))) fail("lua.unsupported_parameter", location);
    const signature = validateAnnotations(annotations, parameters, location, records,protocolTypes);
    annotations = [];
    index += 1;
    const body = []; let depth = 1;
    while (index < lines.length && depth > 0) {
      const item = lines[index].trim();
      if (/^(?:while\s+.+\s+do|if\s+.+\s+then|for\s+.+\s+do)$/.test(item) || !/^end,\s*function/.test(item)&&/function\([A-Za-z_]\w*\)\s*$/.test(item)) depth += 1;
      if (item === "end" || item === "end)") depth -= 1;
      if (depth > 0) body.push({ text: lines[index], line: index + 1 });
      index += 1;
    }
    if (depth > 0) fail("lua.unclosed_function", location);
    const meaningful = body.filter((item) => item.text.trim());
    if (meaningful.length === 1) {
      const returned = /^\s*return\s+(.+?)\s*$/.exec(meaningful[0].text);
      if (!returned) fail("lua.requires_return", { file, line: meaningful[0].line, column: 1 });
      declarations.push({ name: match[2], identity: signature.identity, parameters: signature.parameters, resultType: signature.resultType, expression: parseExpression(returned[1], file, meaningful[0].line), exported: !match[1], location });
    } else declarations.push({ name: match[2], identity: signature.identity, parameters: signature.parameters, resultType: signature.resultType, statements: parseControlBlock(meaningful, file), exported: !match[1], location });
  }
  if (annotations.length) fail("lua.orphan_annotation", { file, line: annotations[0].line, column: 1 });
  return declarations;
}

function parseControlBlock(lines, file, start = 0, nested = false) {
  const statements = []; let index = start;
  while (index < lines.length) {
    const text = lines[index].text.trim(), location = { file, line: lines[index].line, column: 1 };
    if (text === "end" || text === "end)"||text==="else") return { statements, next: index + 1, terminator: text };
    const resultSeparator=/^end,\s*function\(([A-Za-z_]\w*)\)$/.exec(text);if(resultSeparator)return{statements,next:index+1,terminator:"result-separator",binding:resultSeparator[1]};
    let optionCallback = /^return\s+Seme\.match_option\((.*),\s*function\(([A-Za-z_]\w*)\)\s*$/.exec(text);
    if (optionCallback) {
      const arms = splitCallArguments(optionCallback[1], file, lines[index].line);
      if (arms.length !== 2) fail("lua.option_match_arity", location);
      const child = parseControlBlock(lines, file, index + 1, true);
      if (child.terminator !== "end)") fail("lua.option_match_callback_end", location);
      statements.push({ kind: "return_option_match_block", value: parseExpression(arms[0], file, lines[index].line), none: parseExpression(arms[1], file, lines[index].line), binding: optionCallback[2], body: child.statements, location });
      index = child.next;
      continue;
    }
    const resultCallback=/^return\s+Seme\.match_result\((.*),\s*function\(([A-Za-z_]\w*)\)$/.exec(text);
    if(resultCallback){const ok=parseControlBlock(lines,file,index+1,true);if(ok.terminator!=="result-separator")fail("lua.result_match_callback_separator",location);const error=parseControlBlock(lines,file,ok.next,true);if(error.terminator!=="end)")fail("lua.result_match_callback_end",location);statements.push({kind:"return_result_match_block",value:parseExpression(resultCallback[1],file,lines[index].line),okBinding:resultCallback[2],okBody:ok.statements,errorBinding:ok.binding,errorBody:error.statements,location});index=error.next;continue;}
    let closure = /^local\s+([A-Za-z_]\w*)\s*=\s*function\(([A-Za-z_]\w*)\)\s*(?:(return)\s+)?(.+)\s+end$/.exec(text);
    if (closure) {
      if (!closure[3]) {
        const assignment = /^([A-Za-z_]\w*)\s*=\s*(.+)$/.exec(closure[4]);
        if (!assignment) fail("lua.closure_body", location);
        statements.push({ kind:"declare", name:closure[1], expression:{kind:"mutable_closure",parameter:closure[2],capture:assignment[1],body:parseExpression(assignment[2],file,lines[index].line),location}, location });
      } else statements.push({ kind:"declare", name:closure[1], expression:{kind:"immutable_closure",parameter:closure[2],body:parseExpression(closure[4],file,lines[index].line),location}, location });
      index += 1; continue;
    }
    let match = /^local\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+)$/.exec(text);
    if (match) { statements.push({ kind: "declare", name: match[1], expression: parseExpression(match[2], file, lines[index].line), location }); index += 1; continue; }
    match = /^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+)$/.exec(text);
    if (match) { statements.push({ kind: "assign", name: match[1], expression: parseExpression(match[2], file, lines[index].line), location }); index += 1; continue; }
    match = /^while\s+(.+)\s+do$/.exec(text);
    if (match) { const child = parseControlBlock(lines, file, index + 1, true); statements.push({ kind: "while", condition: parseExpression(match[1], file, lines[index].line), body: child.statements, location }); index = child.next; continue; }
    match = /^if\s+(.+)\s+then$/.exec(text);
    if (match) { const child = parseControlBlock(lines, file, index + 1, true);if(child.terminator==="else"){const alternate=parseControlBlock(lines,file,child.next,true);if(alternate.terminator!=="end")fail("lua.if_else_end",location);statements.push({kind:"if_else",condition:parseExpression(match[1],file,lines[index].line),whenTrue:child.statements,whenFalse:alternate.statements,location});index=alternate.next;}else{statements.push({ kind: "when", condition: parseExpression(match[1], file, lines[index].line), body: child.statements, location }); index = child.next;}continue; }
    match = /^return\s+(.+)$/.exec(text);
    if (match) { statements.push({ kind: "return", expression: parseExpression(match[1], file, lines[index].line), location }); index += 1; continue; }
    match=/^Seme\.observe\((.*)\)$/.exec(text);if(match){const argument=match[1].trim(),expression=argument==="true"||argument==="false"?{kind:"bool_literal",value:argument==="true",location}:parseExpression(argument,file,lines[index].line);statements.push({kind:"effect",expression,location});index+=1;continue;}
    match=/^([A-Za-z_]\w*)\((.*)\)$/.exec(text);if(match){const arguments_=splitCallArguments(match[2],file,lines[index].line);statements.push({kind:"call_statement",expression:{kind:"raw_call",name:match[1],arguments:arguments_.map(value=>parseExpression(value,file,lines[index].line)),location},location});index+=1;continue;}
    fail("lua.unsupported_statement", location);
  }
  if (nested) fail("lua.unclosed_block", { file, line: lines.at(-1)?.line ?? 1, column: 1 });
  return statements;
}

function validateAnnotations(annotations, parameters, location, records,protocolTypes) {
  const declared = annotations.filter((item) => item.text.startsWith("---@param ")).map((item) => /^---@param\s+([A-Za-z_][A-Za-z0-9_]*)\s+(\S+)$/.exec(item.text));
  const result = annotations.filter((item) => item.text.startsWith("---@return "));
  const identities = annotations.filter((item) => item.text.startsWith("---@seme-id "));
  const identity = identities.length === 1 ? /^---@seme-id\s+(80[0-9a-f]{30})$/.exec(identities[0].text) : null;
  const returned = result.length === 1 ? /^---@return\s+(\S+)$/.exec(result[0].text) : null;
  if (declared.some((item) => !item) || !returned) fail("lua.unsupported_annotation", location);
  if (identities.length && !identity) fail("lua.invalid_semantic_identity", { ...location, line: identities[0].line });
  if (declared.length !== parameters.length || declared.some((item, index) => item[1] !== parameters[index])) fail("lua.signature_mismatch", location);
  if (annotations.length !== declared.length + 1 + identities.length) fail("lua.unsupported_annotation", location);
  return { identity: identity?.[1], parameters: declared.map((item) => ({ name: item[1], type: annotationType(item[2], location, records,protocolTypes) })), resultType: annotationType(returned[1], location, records,protocolTypes) };
}
function annotationType(value, location, records,protocolTypes=new Map()) {
  const scalar = ({ boolean: "bool", "seme.i64": "i64", "seme.text": "text", "seme.bytes": "bytes" })[value];
  if (scalar) return scalar;
  const option = /^seme\.option<(.+)>$/.exec(value);
  if (option) return `option:${annotationType(option[1], location, records,protocolTypes)}`;
  const protocol=/^seme\.protocol<([A-Za-z_]\w*)>$/.exec(value);if(protocol){const item=protocolTypes.get(protocol[1]);if(!item)fail("lua.unknown_protocol_type",location);return`interface:${item.id}:${item.name}`;}
  if(value==="seme.function<seme.i64,seme.i64>")return"function:i64:i64";
  const transition=/^seme\.transition<([^,]+),([^>]+)>$/.exec(value);if(transition){const record=records.get(transition[1]);if(!record)fail("lua.unsupported_annotation",location);return `transition:${record.id}:${record.name}:${annotationType(transition[2],location,records)}`;}
  const generic = /^(seme\.(?:result|array|slice|map))<(.+)>$/.exec(value);
  if (generic) {
    const arguments_ = splitGenericArguments(generic[2], location);
    if (generic[1] === "seme.result" && arguments_.length === 2) return `result:${annotationType(arguments_[0], location, records,protocolTypes)}:${annotationType(arguments_[1], location, records,protocolTypes)}`;
    if (generic[1] === "seme.array" && arguments_.length === 2 && /^[1-9][0-9]*$/.test(arguments_[1])) return `array:${annotationType(arguments_[0], location, records)}:${arguments_[1]}`;
    if (generic[1] === "seme.slice" && arguments_.length === 1) return `slice:${annotationType(arguments_[0], location, records)}`;
    if (generic[1] === "seme.map" && arguments_.length === 2) return `map:${annotationType(arguments_[0], location, records)}:${annotationType(arguments_[1], location, records)}`;
  }
  if (records.has(value)) return `record:${records.get(value).id}:${value}`;
  fail("lua.unsupported_annotation", location);
}
function splitGenericArguments(value, location) {
  const result = []; let depth = 0; let start = 0;
  for (let index = 0; index < value.length; index += 1) {
    if (value[index] === "<") depth += 1;
    else if (value[index] === ">") { depth -= 1; if (depth < 0) fail("lua.unsupported_annotation", location); }
    else if (value[index] === "," && depth === 0) { result.push(value.slice(start, index)); start = index + 1; }
  }
  if (depth !== 0) fail("lua.unsupported_annotation", location);
  result.push(value.slice(start));
  return result;
}

function parseExpression(text, file, line) {
  text=text.trim();
  const explicitClosure=/^Seme\.(mutable_closure|closure)\((.*),\s*function\(([A-Za-z_]\w*),\s*([A-Za-z_]\w*)\)\s*return\s*(.*)\s*end\)$/.exec(text);if(explicitClosure)return{kind:"canonical_closure",mutable:explicitClosure[1]==="mutable_closure",initial:parseExpression(explicitClosure[2],file,line),capture:explicitClosure[3],parameter:explicitClosure[4],body:parseExpression(explicitClosure[5],file,line),location:{file,line,column:1}};
  const boxed=/^Seme\.box_protocol\(\s*([A-Za-z_]\w*)\s*,\s*(.*)\)$/.exec(text);if(boxed)return{kind:"interface_value",implementation:boxed[1],value:parseExpression(boxed[2],file,line),location:{file,line,column:1}};
  const protocolCall=/^Seme\.protocol_call\((.*)\)$/.exec(text);if(protocolCall){const values=splitCallArguments(protocolCall[1],file,line);if(values.length!==3)fail("lua.protocol_call_arity",{file,line,column:1});const requirement=/^"([A-Za-z_]\w*)"$/.exec(values[1]);if(!requirement)fail("lua.protocol_call_requirement",{file,line,column:1});return{kind:"protocol_call",receiver:parseExpression(values[0],file,line),requirement:requirement[1],arguments:[parseExpression(values[2],file,line)],location:{file,line,column:1}};}
  const resultHelper=/^Seme\.(check_positive|increment_positive)\(\s*([A-Za-z_]\w*)\s*\)$/.exec(text);if(resultHelper)return{kind:resultHelper[1],value:resultHelper[2],location:{file,line,column:1}};
  const transition=/^Seme\.transition_step\(\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*,\s*"([A-Za-z_]\w*)"\s*\)$/.exec(text);if(transition)return{kind:"transition_step",value:transition[1],delta:transition[2],field:transition[3],location:{file,line,column:1}};
  const immutableClosure=/^Seme\.immutable_closure_run\(\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*\)$/.exec(text);if(immutableClosure)return{kind:"immutable_closure_run",base:immutableClosure[1],value:immutableClosure[2],location:{file,line,column:1}};
  const mutableClosure=/^Seme\.mutable_closure_run\(\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*\)$/.exec(text);if(mutableClosure)return{kind:"mutable_closure_run",start:mutableClosure[1],first:mutableClosure[2],second:mutableClosure[3],location:{file,line,column:1}};
  const dispatch = /^Seme\.protocol_dispatch\(\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*,\s*"([A-Za-z_]\w*)"\s*,\s*"([A-Za-z_]\w*)"\s*,\s*"([A-Za-z_]\w*)"\s*,\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*\)$/.exec(text);
  if (dispatch) return { kind: "protocol_dispatch", condition: dispatch[1], whenFalse: dispatch[2], whenTrue: dispatch[3], requirement: dispatch[4], record: dispatch[5], field: dispatch[6], receiver: dispatch[7], argument: dispatch[8], location: { file, line, column: 1 } };
  const scalarDispatch = /^Seme\.protocol_dispatch_value\(\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*,\s*"([A-Za-z_]\w*)"\s*,\s*([A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*\)$/.exec(text);
  if (scalarDispatch) return { kind: "protocol_dispatch_value", condition: scalarDispatch[1], whenFalse: scalarDispatch[2], whenTrue: scalarDispatch[3], requirement: scalarDispatch[4], receiver: scalarDispatch[5], argument: scalarDispatch[6], location: { file, line, column: 1 } };
  const booleanBoundary = /^Seme\.boolean\(([A-Za-z_][A-Za-z0-9_]*)\)$/.exec(text);
  if (booleanBoundary) return { kind:"boolean_boundary", name:booleanBoundary[1], location:{file,line,column:1} };
  const andAt = findTopLevelOperator(text, " and ");
  if (andAt >= 0) return { kind: "boolean_and", left: parseExpression(text.slice(0, andAt), file, line), right: parseExpression(text.slice(andAt + 5), file, line), location: { file, line, column: 1 } };
  const orAt = findTopLevelOperator(text, " or ");
  if (orAt >= 0) return { kind: "boolean_or", left: parseExpression(text.slice(0, orAt), file, line), right: parseExpression(text.slice(orAt + 4), file, line), location: { file, line, column: 1 } };
  for (const [name, kind] of [["less_equal", "less_equal"], ["equal_i64", "equal_i64"]]) {
    const match = new RegExp(`^Seme\\.${name}\\((.*)\\)$`).exec(text);
    if (match) { const values = splitCallArguments(match[1], file, line); if (values.length !== 2) fail(`lua.${name}_arity`, { file, line, column: 1 }); return { kind, left: parseExpression(values[0], file, line), right: parseExpression(values[1], file, line), location: { file, line, column: 1 } }; }
  }
  const integer = /^Seme\.i64_literal\("(-?[0-9]+)"\)$/.exec(text);
  if (integer) return { kind: "integer_literal", value: integer[1], location: { file, line, column: 1 } };
  const concat = /^Seme\.text_concat\((.*)\)$/.exec(text);
  if (concat) {
    const arguments_ = splitCallArguments(concat[1], file, line);
    if (arguments_.length !== 2) fail("lua.text_concat_arity", { file, line, column: 1 });
    return { kind: "text_concat", left: parseExpression(arguments_[0], file, line), right: parseExpression(arguments_[1], file, line), location: { file, line, column: 1 } };
  }
  const string = /^"([^"\\]*)"$/.exec(text);
  if (string) return { kind: "text_literal", value: string[1], location: { file, line, column: 1 } };
  const add = /^Seme\.add\((.*)\)$/.exec(text);
  if (add) {
    const arguments_ = splitCallArguments(add[1], file, line);
    if (arguments_.length !== 2) fail("lua.add_arity", { file, line, column: 1 });
    return { kind: "add", left: parseExpression(arguments_[0], file, line), right: parseExpression(arguments_[1], file, line), location: { file, line, column: 1 } };
  }
  const multiply = /^Seme\.multiply\((.*)\)$/.exec(text);
  if (multiply) {
    const arguments_ = splitCallArguments(multiply[1], file, line);
    if (arguments_.length !== 2) fail("lua.multiply_arity", { file, line, column: 1 });
    return { kind: "multiply", left: parseExpression(arguments_[0], file, line), right: parseExpression(arguments_[1], file, line), location: { file, line, column: 1 } };
  }
  const subtract=/^Seme\.subtract\((.*)\)$/.exec(text);if(subtract){const arguments_=splitCallArguments(subtract[1],file,line);if(arguments_.length!==2)fail("lua.subtract_arity",{file,line,column:1});return{kind:"subtract",left:parseExpression(arguments_[0],file,line),right:parseExpression(arguments_[1],file,line),location:{file,line,column:1}};}
  const nestedFold=/^Seme\.fold\((.*),\s*([A-Za-z_][A-Za-z0-9_]*),\s*function\(([A-Za-z_][A-Za-z0-9_]*),\s*([A-Za-z_][A-Za-z0-9_]*)\)\s*return\s*Seme\.add\(\3,\s*\4\)\s*end\)$/.exec(text);
  if(nestedFold&&nestedFold[1].includes("Seme."))return{kind:"nested_fold",arguments:[parseExpression(nestedFold[1],file,line),parseExpression(nestedFold[2],file,line)],accumulator:nestedFold[3],element:nestedFold[4],location:{file,line,column:1}};
  const nestedLookup=/^Seme\.lookup_zero\((.*),\s*([A-Za-z_][A-Za-z0-9_]*),\s*"(i64|text|bytes)"\)$/.exec(text);
  if(nestedLookup&&nestedLookup[1].includes("Seme."))return{kind:"nested_lookup_zero",arguments:[parseExpression(nestedLookup[1],file,line),parseExpression(nestedLookup[2],file,line)],descriptor:nestedLookup[3],location:{file,line,column:1}};
  const nested=/^Seme\.(slice|collection_append|collection_update|slice_remove|length|index_zero|map_update|map_remove)\((.*)\)$/.exec(text);
  if(nested&&nested[2].includes("Seme.")){const arguments_=splitCallArguments(nested[2],file,line).map(value=>parseExpression(value,file,line));return{kind:`nested_${nested[1]}`,arguments:arguments_,location:{file,line,column:1}};}
  const composite = /^Seme\.match_option\(([A-Za-z_][A-Za-z0-9_]*), false, function\(([A-Za-z_][A-Za-z0-9_]*)\) return Seme\.match_result\(\2, function\(([A-Za-z_][A-Za-z0-9_]*)\) return Seme\.bytes_equal\(\3, Seme\.bytes_literal\("([^"]*)"\)\) end, function\(([A-Za-z_][A-Za-z0-9_]*)\) return Seme\.text_equal\(\5, "([^"]*)"\) end\) end\)$/.exec(text);
  if (composite) return { kind: "composite_match", value: composite[1], some: composite[2], ok: composite[3], bytes: composite[4], error: composite[5], text: composite[6], location: { file, line, column: 1 } };
  const optionMatch=/^Seme\.match_option\((.*),\s*([^,]+),\s*function\(([A-Za-z_]\w*)\)\s*return\s*(.*)\s*end\)$/.exec(text);
  if(optionMatch)return{kind:"option_match",value:parseExpression(optionMatch[1].trim(),file,line),none:parseExpression(optionMatch[2].trim(),file,line),binding:optionMatch[3],some:parseExpression(optionMatch[4].trim(),file,line),location:{file,line,column:1}};
  const resultMatch=/^Seme\.match_result\((.*),\s*function\(([A-Za-z_]\w*)\)\s*return\s*(.*)\s*end,\s*function\(([A-Za-z_]\w*)\)\s*return\s*(.*)\s*end\)$/.exec(text);
  if(resultMatch)return{kind:"result_match",value:parseExpression(resultMatch[1].trim(),file,line),okBinding:resultMatch[2],ok:parseExpression(resultMatch[3].trim(),file,line),errorBinding:resultMatch[4],error:parseExpression(resultMatch[5].trim(),file,line),location:{file,line,column:1}};
  const recordConstruct=/^Seme\.record\("([A-Za-z_]\w*)",\s*\{([^}]*)\},\s*\{([^}]*)\}\)$/.exec(text);
  if(recordConstruct){const names=recordConstruct[2].trim()?recordConstruct[2].split(",").map(item=>{const match=/^\s*"([A-Za-z_]\w*)"\s*$/.exec(item);if(!match)fail("lua.record_construct_fields",{file,line,column:1});return match[1]}):[],values=splitCallArguments(recordConstruct[3],file,line).map(item=>{const match=/^([A-Za-z_]\w*)\s*=\s*(.*)$/.exec(item);if(!match)fail("lua.record_construct_fields",{file,line,column:1});return{name:match[1],value:parseExpression(match[2],file,line)}});if(names.length!==values.length||names.some((name,index)=>name!==values[index].name))fail("lua.record_construct_fields",{file,line,column:1});return{kind:"record_construct",name:recordConstruct[1],fields:values,location:{file,line,column:1}};}
  const transitionConstruct=/^Seme\.transition\((.*)\)$/.exec(text);
  if(transitionConstruct){const arguments_=splitCallArguments(transitionConstruct[1],file,line).map(item=>parseExpression(item,file,line));return{kind:"transition_construct",arguments:arguments_,location:{file,line,column:1}};}
  const rawCall=/^([A-Za-z_]\w*)\((.*)\)$/.exec(text);
  if(rawCall)return{kind:"raw_call",name:rawCall[1],arguments:rawCall[2].trim()?splitCallArguments(rawCall[2],file,line).map(item=>parseExpression(item,file,line)):[],location:{file,line,column:1}};
  const field = /^Seme\.field\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*"([A-Za-z_][A-Za-z0-9_]*)"\s*\)$/.exec(text);
  if (field) return { kind: "field", base: field[1], field: field[2], location: { file, line, column: 1 } };
  const array = /^Seme\.array\s*\((.*)\)$/.exec(text);
  if (array) return { kind: "array", arguments: array[1].trim() ? array[1].split(",").map((name) => name.trim()) : [], location: { file, line, column: 1 } };
  const sliceConstruct=/^Seme\.slice\s*\((.*)\)$/.exec(text);
  if(sliceConstruct)return{kind:"slice",arguments:sliceConstruct[1].trim()?splitCallArguments(sliceConstruct[1],file,line):[],location:{file,line,column:1}};
  const length = /^Seme\.(?:length|length_i64)\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)$/.exec(text);
  if (length) return { kind: "length", base: length[1], location: { file, line, column: 1 } };
  const index = /^Seme\.index_zero\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*|[0-9]+)\s*\)$/.exec(text);
  if (index) return { kind: "index", base: index[1], index: index[2], location: { file, line, column: 1 } };
  const emptyMap = /^Seme\.empty_map\s*\(\s*"(i64|text|bytes)"\s*\)$/.exec(text);
  if (emptyMap) return { kind: "empty_map", descriptor: emptyMap[1], location: { file, line, column: 1 } };
  const lookup = /^Seme\.lookup_zero\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*"(i64|text|bytes)"\s*\)$/.exec(text);
  if (lookup) return { kind: "lookup_zero", base: lookup[1], key: lookup[2], descriptor: lookup[3], location: { file, line, column: 1 } };
  const mapLookup = /^Seme\.map_lookup\s*\((.*)\)$/.exec(text);
  if (mapLookup) { const arguments_=splitCallArguments(mapLookup[1],file,line);if(arguments_.length!==2)fail("lua.map_lookup_arity",{file,line,column:1});return { kind: "map_lookup", baseExpression:parseExpression(arguments_[0],file,line), keyExpression:parseExpression(arguments_[1],file,line), location: { file, line, column: 1 } }; }
  const mapOperation = /^Seme\.(map_update)\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)$/.exec(text);
  if (mapOperation) return { kind: mapOperation[1], base: mapOperation[2], key: mapOperation[3], value: mapOperation[4], location: { file, line, column: 1 } };
  const collectionOperation=/^Seme\.(collection_append|collection_update|slice_remove)\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)(?:\s*,\s*([A-Za-z_][A-Za-z0-9_]*))?\s*\)$/.exec(text);
  if(collectionOperation)return{kind:collectionOperation[1],base:collectionOperation[2],index:collectionOperation[1]==="collection_append"?undefined:collectionOperation[3],value:collectionOperation[1]==="collection_append"?collectionOperation[3]:collectionOperation[4],location:{file,line,column:1}};
  const mapRemove=/^Seme\.map_remove\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)$/.exec(text);
  if(mapRemove)return{kind:"map_remove",base:mapRemove[1],key:mapRemove[2],location:{file,line,column:1}};
  const fold=/^Seme\.fold\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*function\(([A-Za-z_][A-Za-z0-9_]*),\s*([A-Za-z_][A-Za-z0-9_]*)\)\s*return\s*Seme\.add\(\3,\s*\4\)\s*end\s*\)$/.exec(text);
  if(fold)return{kind:"fold",base:fold[1],initial:fold[2],accumulator:fold[3],element:fold[4],location:{file,line,column:1}};
  const identifier = /^([A-Za-z_][A-Za-z0-9_]*)$/.exec(text);
  if (identifier) return { kind: "identifier", name: identifier[1], location: { file, line, column: 1 } };
  const constructor = /^Seme\.(some|none|ok|err)\s*\((.*)\)$/.exec(text);
  if (constructor) {
    const arguments_ = constructor[2].trim() ? [parseExpression(constructor[2].trim(), file, line)] : [];
    return { kind: constructor[1], arguments: arguments_, location: { file, line, column: 1 } };
  }
  const call = /^([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)$/.exec(text);
  if (call) {
    const arguments_ = call[2].trim() ? call[2].split(",").map((item) => parseExpression(item.trim(), file, line)) : [];
    return { kind: "call", name: call[1], arguments: arguments_, location: { file, line, column: 1 } };
  }
  fail("lua.unsupported_expression", { file, line, column: 1 });
}

function findTopLevelOperator(value, operator) { let depth = 0, quoted = false; for (let index = 0; index <= value.length - operator.length; index += 1) { const c = value[index]; if (c === '"' && value[index - 1] !== "\\") quoted = !quoted; else if (!quoted && c === "(") depth += 1; else if (!quoted && c === ")") depth -= 1; if (!quoted && depth === 0 && value.slice(index, index + operator.length) === operator) return index; } return -1; }

function composableSymbol(name,context,location){const local=context.symbols?.get(name);if(local)return local;const index=context.description.parameters.findIndex(item=>item.name===name);if(index<0)fail("lua.unknown_identifier",location);return{kind:"parameter",type:context.description.parameters[index].type,id:context.parameterIDs[index]};}
function inferComposableExpression(expression,context){
  if(expression.kind==="identifier")return composableSymbol(expression.name,context,expression.location).type;
  if(expression.kind==="field"){
    const base=inferComposableExpression({kind:"identifier",name:expression.base,location:expression.location},context);if(!base.startsWith("record:"))fail("lua.field_receiver_type",expression.location);
    const record=context.records.get(base.split(":").slice(2).join(":")),member=record?.fields.find(item=>item.name===expression.field);if(!member)fail("lua.unknown_field",expression.location);return member.type;
  }
  if(expression.kind==="map_lookup"){
    const map=inferComposableExpression(expression.baseExpression,context),key=inferComposableExpression(expression.keyExpression,context);if(!map.startsWith("map:"))fail("lua.map_receiver_type",expression.location);const[,keyType,valueType]=map.split(":");if(key!==keyType)fail("lua.map_key_type",expression.location);return`option:${valueType}`;
  }
  fail("lua.composable_expression_profile",expression.location);
}
function emitComposableExpression(expression,context,path){
  const type=inferComposableExpression(expression,context);
  if(expression.kind==="identifier"){const symbol=composableSymbol(expression.name,context,expression.location),id=stableID("execution",context.description.id,path,"symbol-read"),shape=symbol.kind==="parameter"?[schema.read,0x9130]:symbol.kind==="variant"?[schema.variantRead,0xa0610]:symbol.kind==="local"?[schema.localRead,0x9d20]:[schema.placeRead,0x9e20];context.additions.push(graphEntity(id,entity(id,shape[0],[[shape[1],ref(symbol.id)]])));return{id,type};}
  if(expression.kind==="field"){
    const base=emitComposableExpression({kind:"identifier",name:expression.base,location:expression.location},context,`${path}.base`),record=context.records.get(base.type.split(":").slice(2).join(":")),index=record.fields.findIndex(item=>item.name===expression.field),id=stableID("execution",context.description.id,path,"field-read");context.additions.push(graphEntity(id,entity(id,schema.fieldRead,[[0x9320,ref(base.id)],[0x9321,ref(stableID("execution",record.id,"field",String(index)))]])));return{id,type};
  }
  const map=emitComposableExpression(expression.baseExpression,context,`${path}.map`),key=emitComposableExpression(expression.keyExpression,context,`${path}.key`),id=stableID("execution",context.description.id,path,"map-lookup-option");ensureType(type,context.additions,context.records);context.additions.push(graphEntity(id,entity(id,schema.mapLookupOption,[[0xa0440,ref(map.id)],[0xa0441,ref(key.id)],[0xa0442,ref(typeID(type))]])));return{id,type};
}

function emitExpression(expression, context, path) {
  if(expression.kind==="canonical_closure"||expression.kind==="protocol_call"||expression.kind==="interface_value")return emitControlExpression(expression,context.description.resultType,{...context,symbols:new Map(context.description.parameters.map((item,index)=>[item.name,{kind:"parameter",type:item.type,id:context.parameterIDs[index]}]))},path);
  if(expression.kind==="record_construct"||expression.kind==="transition_construct"||((expression.kind==="ok"||expression.kind==="err")&&expression.arguments[0]?.kind!=="identifier"))return emitControlExpression(expression,context.description.resultType,{...context,symbols:new Map(context.description.parameters.map((item,index)=>[item.name,{kind:"parameter",type:item.type,id:context.parameterIDs[index]}]))},path);
  if(expression.kind.startsWith("nested_"))return emitNestedExpression(expression,context.description.resultType,context,path);
  if (expression.kind === "identifier") {
    if (context.receiver?.name === expression.name) {
      if (context.receiver.type !== context.description.resultType) fail("lua.receiver_return_type", expression.location);
      const id = stableID("execution", context.description.id, "expression", path, "receiver-read");
      context.additions.push(graphEntity(id, entity(id, schema.receiverRead, [[0xa0010, ref(context.receiver.id)]])));
      return id;
    }
    const index = context.description.parameters.findIndex((parameter) => parameter.name === expression.name);
    if (index < 0) fail("lua.unknown_identifier", expression.location);
    if (context.description.parameters[index].type !== context.description.resultType) fail("lua.return_type", expression.location);
    const id = stableID("execution", context.description.id, "expression", path, "parameter-read");
    context.additions.push(graphEntity(id, entity(id, schema.read, [[0x9130, ref(context.parameterIDs[index])]])));
    return id;
  }
  if (["some", "none", "ok", "err"].includes(expression.kind)) return emitConstructor(expression, context, path);
  if (expression.kind === "field") {
    if (context.receiver?.name === expression.base) {
      const record = context.records.get(context.receiver.type.split(":").slice(2).join(":"));
      const fieldIndex = record?.fields.findIndex((item) => item.name === expression.field) ?? -1;
      if (fieldIndex < 0 || record.fields[fieldIndex].type !== context.description.resultType) fail("lua.field_result_type", expression.location);
      const readID = stableID("execution", context.description.id, "expression", `${path}.base`, "receiver-read"), id = stableID("execution", context.description.id, "expression", path, "field-read");
      context.additions.push(graphEntity(readID, entity(readID, schema.receiverRead, [[0xa0010, ref(context.receiver.id)]])));
      context.additions.push(graphEntity(id, entity(id, schema.fieldRead, [[0x9320, ref(readID)], [0x9321, ref(stableID("execution", record.id, "field", String(fieldIndex)))]])));
      return id;
    }
    const index = context.description.parameters.findIndex((parameter) => parameter.name === expression.base);
    if (index < 0) fail("lua.unknown_identifier", expression.location);
    const parameterType = context.description.parameters[index].type;
    if (!parameterType.startsWith("record:")) fail("lua.field_receiver_type", expression.location);
    const record = context.records.get(parameterType.split(":").slice(2).join(":"));
    const fieldIndex = record?.fields.findIndex((field) => field.name === expression.field) ?? -1;
    if (fieldIndex < 0) fail("lua.unknown_field", expression.location);
    if (record.fields[fieldIndex].type !== context.description.resultType) fail("lua.field_result_type", expression.location);
    const readID = stableID("execution", context.description.id, "expression", `${path}.base`, "parameter-read");
    const id = stableID("execution", context.description.id, "expression", path, "field-read");
    context.additions.push(graphEntity(readID, entity(readID, schema.read, [[0x9130, ref(context.parameterIDs[index])]])));
    const fieldID = stableID("execution", record.id, "field", String(fieldIndex));
    context.additions.push(graphEntity(id, entity(id, schema.fieldRead, [[0x9320, ref(readID)], [0x9321, ref(fieldID)]])));
    return id;
  }
  if(expression.kind==="map_lookup"){
    const inferred=inferComposableExpression(expression,context);
    if(inferred!==context.description.resultType)fail("lua.map_lookup_option_type",expression.location);
    return emitComposableExpression(expression,context,path).id;
  }
  if (["array", "slice", "length", "index", "fold", "empty_map", "lookup_zero", "map_update", "collection_append", "collection_update", "slice_remove", "map_remove"].includes(expression.kind)) return emitCollectionExpression(expression, context, path);
  if (expression.kind === "composite_match") return emitCompositeMatch(expression, context, path);
  if(expression.kind==="option_match")return emitOptionMatch(expression,context,path);
  if(expression.kind==="result_match")return emitResultMatch(expression,context,path);
  if (expression.kind === "add") {
    if (context.description.resultType !== "i64") fail("lua.add_result_type", expression.location);
    const left = emitExpression(expression.left, context, `${path}.left`);
    const right = emitExpression(expression.right, context, `${path}.right`);
    const id = stableID("execution", context.description.id, "expression", path, "add");
    context.additions.push(graphEntity(id, entity(id, schema.integerAdd, [[0x9140, ref(left)], [0x9141, ref(right)], [0x9142, ref(ids.i64)]])));
    return id;
  }
  if (expression.kind === "multiply") {
    if (context.description.resultType !== "i64") fail("lua.multiply_result_type", expression.location);
    const left = emitExpression(expression.left, context, `${path}.left`);
    const right = emitExpression(expression.right, context, `${path}.right`);
    const id = stableID("execution", context.description.id, "expression", path, "multiply");
    context.additions.push(graphEntity(id, entity(id, schema.integerMultiply, [[0x9900, ref(left)], [0x9901, ref(right)], [0x9902, ref(ids.i64)]])));
    return id;
  }
  if (expression.kind === "integer_literal") {
    if (context.description.resultType !== "i64") fail("lua.integer_literal_result_type", expression.location);
    const id = stableID("execution", context.description.id, "expression", path, "integer-literal");
    context.additions.push(graphEntity(id, entity(id, schema.integerLiteral, [[0x9700, `uu ${i64Word(expression.value,expression.location)}`], [0x9701, ref(ids.i64)]])));
    return id;
  }
  if (expression.kind === "text_literal") {
    if (context.description.resultType !== "text") fail("lua.text_literal_result_type", expression.location);
    const id = stableID("execution", context.description.id, "expression", path, "text-literal");
    context.additions.push(graphEntity(id, entity(id, schema.stringLiteral, [[0x9500, bytes(expression.value)]])));
    return id;
  }
  if (expression.kind === "text_concat") {
    if (context.description.resultType !== "text") fail("lua.text_concat_result_type", expression.location);
    const left = emitExpression(expression.left, context, `${path}.left`), right = emitExpression(expression.right, context, `${path}.right`);
    const id = stableID("execution", context.description.id, "expression", path, "text-concat");
    context.additions.push(graphEntity(id, entity(id, schema.stringConcat, [[0x9c30, ref(left)], [0x9c31, ref(right)]])));
    return id;
  }
  const callee = context.descriptionsByName.get(expression.name);
  if (!callee) fail("lua.unknown_call", expression.location);
  if (callee.parameters.length !== expression.arguments.length) fail("lua.call_arity", expression.location);
  if (callee.resultType !== context.description.resultType) fail("lua.call_result_type", expression.location);
  for (let index = 0; index < expression.arguments.length; index += 1) {
    const argument = expression.arguments[index];
    const actual=argument.kind==="identifier"?context.description.parameters.find(parameter=>parameter.name===argument.name)?.type:argument.kind==="slice"?"slice:i64":null;
    if (!actual || actual !== callee.parameters[index].type) fail("lua.call_argument_type", argument.location);
  }
  const arguments_ = expression.arguments.map((item, index) => emitExpression(item, {...context,description:{...context.description,resultType:callee.parameters[index].type}}, `${path}.argument.${index}`));
  const id = stableID("execution", context.description.id, "expression", path, "call");
  context.additions.push(graphEntity(id, entity(id, schema.call, [[0x9600, ref(callee.id)], [0x9601, refs(arguments_)]])));
  return id;
}

function emitNestedExpression(expression,expected,context,path){
  // Nested expressions can introduce collection types that are absent from the
  // public function signature. Materialize those types as part of the graph;
  // a reference to a stable type identity is never sufficient on its own.
  ensureType(expected, context.additions, context.records);
  const kind=expression.kind.slice(7),args=expression.arguments,id=stableID("execution",context.description.id,"expression",path,kind);
  const emit=(value,type,suffix)=>value.kind.startsWith("nested_")?emitNestedExpression(value,type,context,`${path}.${suffix}`):value.kind==="identifier"?parameterRead(value.name,type,context,`${path}.${suffix}`,value.location):(()=>{const nestedContext={...context,description:{...context.description,resultType:type}};return emitExpression(value,nestedContext,`${path}.${suffix}`)})();
  if(kind==="fold"){if(expected!=="i64"||args.length!==2)fail("lua.fold_result_type",expression.location);const collection=emit(args[0],"slice:i64","collection"),initial=emit(args[1],"i64","initial"),accumulator=stableID("execution",context.description.id,path,"accumulator"),element=stableID("execution",context.description.id,path,"element"),left=stableID("execution",context.description.id,path,"left"),right=stableID("execution",context.description.id,path,"right"),body=stableID("execution",context.description.id,path,"body");context.additions.push(graphEntity(accumulator,entity(accumulator,schema.iterationBinding,[[0x9f50,bytes(expression.accumulator)],[0x9f51,ref(ids.i64)]])),graphEntity(element,entity(element,schema.iterationBinding,[[0x9f50,bytes(expression.element)],[0x9f51,ref(ids.i64)]])),graphEntity(left,entity(left,schema.iterationRead,[[0x9f60,ref(accumulator)]])),graphEntity(right,entity(right,schema.iterationRead,[[0x9f60,ref(element)]])),graphEntity(body,entity(body,schema.integerAdd,[[0x9140,ref(left)],[0x9141,ref(right)],[0x9142,ref(ids.i64)]])),graphEntity(id,entity(id,schema.fold,[[0x9f70,ref(collection)],[0x9f71,ref(initial)],[0x9f72,ref(accumulator)],[0x9f73,ref(element)],[0x9f74,ref(body)]])));return id;}
  if(kind==="lookup_zero"){if(expected!==expression.descriptor||args.length!==2)fail("lua.map_lookup_type",expression.location);const mapType=`map:i64:${expected}`,map=emit(args[0],mapType,"map"),key=emit(args[1],"i64","key");context.additions.push(graphEntity(id,entity(id,schema.mapLookup,[[0xa0420,ref(map)],[0xa0421,ref(key)]])));return id;}
  if(kind==="slice"){if(expected!=="slice:i64"||args.length>512)fail("lua.slice_construct_type_or_bound",expression.location);const values=args.map((value,index)=>emit(value,"i64",`element.${index}`));context.additions.push(graphEntity(id,entity(id,schema.sliceConstruct,[[0xa0680,ref(typeID(expected))],[0xa0681,refs(values)]])));return id;}
  if(kind==="collection_append"){if(expected!=="slice:i64"||args.length!==2)fail("lua.collection_append_type",expression.location);const base=emit(args[0],expected,"collection"),value=emit(args[1],"i64","value");context.additions.push(graphEntity(id,entity(id,schema.collectionAppend,[[0x9fb0,ref(base)],[0x9fb1,ref(value)]])));return id;}
  if(kind==="collection_update"){if(expected!=="slice:i64"||args.length!==3)fail("lua.collection_update_type",expression.location);const base=emit(args[0],expected,"collection"),index=emit(args[1],"i64","index"),value=emit(args[2],"i64","value");context.additions.push(graphEntity(id,entity(id,schema.collectionUpdate,[[0x9fc0,ref(base)],[0x9fc1,ref(index)],[0x9fc2,ref(value)]])));return id;}
  if(kind==="slice_remove"){if(expected!=="slice:i64"||args.length!==2)fail("lua.slice_remove_type",expression.location);const base=emit(args[0],expected,"collection"),index=emit(args[1],"i64","index");context.additions.push(graphEntity(id,entity(id,schema.sliceRemove,[[0xa0660,ref(base)],[0xa0661,ref(index)]])));return id;}
  if(kind==="length"||kind==="index_zero"){if(expected!=="i64"||(kind==="length"?args.length!==1:args.length!==2))fail("lua.collection_query_type",expression.location);const base=emit(args[0],"slice:i64","collection");if(kind==="length"){context.additions.push(graphEntity(id,entity(id,schema.collectionLength,[[0x9f90,ref(base)]])));return id;}const index=emit(args[1],"i64","index");context.additions.push(graphEntity(id,entity(id,schema.dynamicIndexRead,[[0x9fa0,ref(base)],[0x9fa1,ref(index)]])));return id;}
  if(kind==="map_update"||kind==="map_remove"){if(!expected.startsWith("map:")||args.length!==(kind==="map_update"?3:2))fail("lua.map_mutation_type",expression.location);const[,keyType,valueType]=expected.split(":"),base=emit(args[0],expected,"map"),key=emit(args[1],keyType,"key");if(kind==="map_remove"){context.additions.push(graphEntity(id,entity(id,schema.mapRemove,[[0xa0670,ref(base)],[0xa0671,ref(key)]])));return id;}const value=emit(args[2],valueType,"value");context.additions.push(graphEntity(id,entity(id,schema.mapUpdate,[[0xa0430,ref(base)],[0xa0431,ref(key)],[0xa0432,ref(value)]])));return id;}
  fail("lua.nested_expression_profile",expression.location);
}

function splitCallArguments(value, file, line) {
  const result = []; let depth = 0; let quoted = false; let start = 0;
  for (let index = 0; index < value.length; index += 1) {
    const char = value[index];
    if (char === '"' && value[index - 1] !== "\\") quoted = !quoted;
    else if (!quoted && char === "(") depth += 1;
    else if (!quoted && char === ")") depth -= 1;
    else if (!quoted && depth === 0 && char === ",") { result.push(value.slice(start, index).trim()); start = index + 1; }
    if (depth < 0) fail("lua.call_parentheses", { file, line, column: 1 });
  }
  if (quoted || depth !== 0) fail("lua.call_parentheses", { file, line, column: 1 });
  result.push(value.slice(start).trim()); return result;
}

function emitArmValue(expression, expected, context, path) {
  if (expression.kind === "identifier") {
    const variant=context.variants?.get(expression.name);
    if(variant){if(variant.type!==expected)fail("lua.match_arm_type",expression.location);const id=stableID("execution",context.description.id,path,"variant-read");context.additions.push(graphEntity(id,entity(id,schema.variantRead,[[0xa0610,ref(variant.id)]])));return id;}
    return parameterRead(expression.name,expected,context,path,expression.location);
  }
  if(expression.kind==="integer_literal"){
    if(expected!=="i64")fail("lua.match_arm_type",expression.location);const id=stableID("execution",context.description.id,path,"integer-literal");context.additions.push(graphEntity(id,entity(id,schema.integerLiteral,[[0x9700,`uu ${i64Word(expression.value,expression.location)}`],[0x9701,ref(ids.i64)]])));return id;
  }
  if(expression.kind==="add"||expression.kind==="multiply"){
    if(expected!=="i64")fail("lua.match_arm_type",expression.location);const left=emitArmValue(expression.left,"i64",context,`${path}.left`),right=emitArmValue(expression.right,"i64",context,`${path}.right`),id=stableID("execution",context.description.id,path,expression.kind),shape=expression.kind==="add"?[schema.integerAdd,0x9140,0x9141,0x9142]:[schema.integerMultiply,0x9900,0x9901,0x9902];context.additions.push(graphEntity(id,entity(id,shape[0],[[shape[1],ref(left)],[shape[2],ref(right)],[shape[3],ref(ids.i64)]])));return id;
  }
  fail("lua.match_arm_expression",expression.location);
}
function returnedExpressionBlock(value, context, path){const returned=stableID("execution",context.description.id,path,"return"),block=stableID("execution",context.description.id,path,"block");context.additions.push(graphEntity(returned,entity(returned,schema.returned,[[0x9810,refs([value])]])),graphEntity(block,entity(block,schema.block,[[0x9800,refs([returned])]])));return block;}
function emitOptionMatch(expression,context,path){
  const optionType=inferComposableExpression(expression.value,context);if(!optionType.startsWith("option:"))fail("lua.option_match_value_type",expression.location);const payloadType=optionType.slice(7),expected=context.description.resultType,value=emitComposableExpression(expression.value,context,`${path}.value`).id,noneValue=emitArmValue(expression.none,expected,context,`${path}.none.value`),noneBlock=returnedExpressionBlock(noneValue,context,`${path}.none`),binding=stableID("execution",context.description.id,path,"some-binding");context.additions.push(graphEntity(binding,entity(binding,schema.variantBinding,[[0xa0600,bytes(expression.binding)],[0xa0601,ref(typeID(payloadType))]])));const someContext={...context,variants:new Map([...(context.variants??[]),[expression.binding,{id:binding,type:payloadType}]])},someValue=emitArmValue(expression.some,expected,someContext,`${path}.some.value`),someBlock=returnedExpressionBlock(someValue,context,`${path}.some`),id=stableID("execution",context.description.id,path,"option-match");context.additions.push(graphEntity(id,entity(id,schema.optionMatch,[[0xa0630,ref(value)],[0xa0631,ref(noneBlock)],[0xa0632,ref(binding)],[0xa0633,ref(someBlock)]])));return id;
}
function emitResultMatch(expression,context,path){
  const resultType=inferComposableExpression(expression.value,context);if(!resultType.startsWith("result:"))fail("lua.result_match_value_type",expression.location);const[okType,errorType]=splitResult(resultType),expected=context.description.resultType,value=emitComposableExpression(expression.value,context,`${path}.value`).id,okBinding=stableID("execution",context.description.id,path,"ok-binding"),errorBinding=stableID("execution",context.description.id,path,"error-binding");context.additions.push(graphEntity(okBinding,entity(okBinding,schema.variantBinding,[[0xa0600,bytes(expression.okBinding)],[0xa0601,ref(typeID(okType))]])),graphEntity(errorBinding,entity(errorBinding,schema.variantBinding,[[0xa0600,bytes(expression.errorBinding)],[0xa0601,ref(typeID(errorType))]])));const okContext={...context,variants:new Map([...(context.variants??[]),[expression.okBinding,{id:okBinding,type:okType}]])},errorContext={...context,variants:new Map([...(context.variants??[]),[expression.errorBinding,{id:errorBinding,type:errorType}]])},okValue=emitArmValue(expression.ok,expected,okContext,`${path}.ok.value`),errorValue=emitArmValue(expression.error,expected,errorContext,`${path}.error.value`),okBlock=returnedExpressionBlock(okValue,context,`${path}.ok`),errorBlock=returnedExpressionBlock(errorValue,context,`${path}.error`),id=stableID("execution",context.description.id,path,"result-match");context.additions.push(graphEntity(id,entity(id,schema.resultMatch,[[0xa0620,ref(value)],[0xa0621,ref(okBinding)],[0xa0622,ref(okBlock)],[0xa0623,ref(errorBinding)],[0xa0624,ref(errorBlock)]])));return id;
}

function emitCompositeMatch(expression, context, path) {
  const expected = "option:result:bytes:text";
  if (context.description.resultType !== "bool") fail("lua.match_result_type", expression.location);
  const value = parameterRead(expression.value, expected, context, `${path}.value`, expression.location);
  const make = (suffix, schemaID, fields) => { const id = stableID("execution", context.description.id, "expression", path, suffix); context.additions.push(graphEntity(id, entity(id, schemaID, fields))); return id; };
  const returnedBlock = (suffix, expressionID) => {
    const returned = make(`${suffix}.return`, schema.returned, [[0x9810, refs([expressionID])]]);
    return make(`${suffix}.block`, schema.block, [[0x9800, refs([returned])]]);
  };
  const noneLiteral = make("none.false", schema.boolLiteral, [[0x9b00, "fa"]]);
  const noneBlock = returnedBlock("none", noneLiteral);
  const resultType = typeID("result:bytes:text");
  const someBinding = make("some.binding", schema.variantBinding, [[0xa0600, bytes(expression.some)], [0xa0601, ref(resultType)]]);
  const someRead = make("some.read", schema.variantRead, [[0xa0610, ref(someBinding)]]);
  const okBinding = make("ok.binding", schema.variantBinding, [[0xa0600, bytes(expression.ok)], [0xa0601, ref(ids.bytes)]]);
  const errorBinding = make("error.binding", schema.variantBinding, [[0xa0600, bytes(expression.error)], [0xa0601, ref(ids.text)]]);
  const okRead = make("ok.read", schema.variantRead, [[0xa0610, ref(okBinding)]]);
  const errorRead = make("error.read", schema.variantRead, [[0xa0610, ref(errorBinding)]]);
  const bytesLiteral = make("ok.literal", schema.bytesLiteral, [[0xa0640, bytes(expression.bytes)]]);
  const textLiteral = make("error.literal", schema.stringLiteral, [[0x9500, bytes(expression.text)]]);
  const bytesEqual = make("ok.equal", schema.bytesEqual, [[0xa0650, ref(okRead)], [0xa0651, ref(bytesLiteral)]]);
  const textEqual = make("error.equal", schema.stringEqual, [[0x9c20, ref(errorRead)], [0x9c21, ref(textLiteral)]]);
  const okBlock = returnedBlock("ok", bytesEqual);
  const errorBlock = returnedBlock("error", textEqual);
  const resultMatch = make("result.match", schema.resultMatch, [[0xa0620, ref(someRead)], [0xa0621, ref(okBinding)], [0xa0622, ref(okBlock)], [0xa0623, ref(errorBinding)], [0xa0624, ref(errorBlock)]]);
  const someBlock = returnedBlock("some", resultMatch);
  return make("option.match", schema.optionMatch, [[0xa0630, ref(value)], [0xa0631, ref(noneBlock)], [0xa0632, ref(someBinding)], [0xa0633, ref(someBlock)]]);
}

function parameterRead(name, expected, context, path, location) {
  const index = context.description.parameters.findIndex((parameter) => parameter.name === name);
  if (index < 0) fail("lua.unknown_identifier", location);
  if (context.description.parameters[index].type !== expected) fail("lua.collection_argument_type", location);
  const id = stableID("execution", context.description.id, "expression", path, "parameter-read");
  context.additions.push(graphEntity(id, entity(id, schema.read, [[0x9130, ref(context.parameterIDs[index])]])));
  return id;
}
function emitCollectionExpression(expression, context, path) {
  const result = context.description.resultType;
  const id = stableID("execution", context.description.id, "expression", path, expression.kind);
  if (expression.kind === "array") {
    if (!result.startsWith("array:")) fail("lua.array_result_type", expression.location);
    const [, element, length] = result.split(":");
    if (expression.arguments.length !== Number(length)) fail("lua.array_arity", expression.location);
    const values = expression.arguments.map((name, index) => parameterRead(name, element, context, `${path}.element.${index}`, expression.location));
    context.additions.push(graphEntity(id, entity(id, schema.fixedArrayConstruct, [[0x9f30, ref(typeID(result))], [0x9f31, refs(values)]])));
    return id;
  }
  if(expression.kind==="slice"){
    if(result!=="slice:i64"||expression.arguments.length>512)fail("lua.slice_construct_type_or_bound",expression.location);
    const values=expression.arguments.map((name,index)=>parameterRead(name,"i64",context,`${path}.element.${index}`,expression.location));
    context.additions.push(graphEntity(id,entity(id,schema.sliceConstruct,[[0xa0680,ref(typeID(result))],[0xa0681,refs(values)]])));return id;
  }
  if (expression.kind === "empty_map") {
    if (!result.startsWith("map:")) fail("lua.map_result_type", expression.location);
    if (result.split(":")[2] !== expression.descriptor) fail("lua.map_descriptor_type", expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.emptyMap, [[0xa0410, ref(typeID(result))]])));
    return id;
  }
  if (expression.kind === "length") {
    if (result !== "i64") fail("lua.length_result_type", expression.location);
    const parameter = context.description.parameters.find((item) => item.name === expression.base);
    if (!parameter || !(parameter.type.startsWith("array:") || parameter.type.startsWith("slice:"))) fail("lua.length_collection_type", expression.location);
    const base = parameterRead(expression.base, parameter.type, context, `${path}.collection`, expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.collectionLength, [[0x9f90, ref(base)]])));
    return id;
  }
  if(expression.kind==="fold"){
    if(result!=="i64")fail("lua.fold_result_type",expression.location);
    const collection=context.description.parameters.find(item=>item.name===expression.base),initial=context.description.parameters.find(item=>item.name===expression.initial);
    if(!collection||!(collection.type==="slice:i64"||collection.type.startsWith("array:i64:"))||!initial||initial.type!=="i64")fail("lua.fold_parameter_type",expression.location);
    const collectionRead=parameterRead(expression.base,collection.type,context,`${path}.collection`,expression.location),initialRead=parameterRead(expression.initial,"i64",context,`${path}.initial`,expression.location);
    const accumulator=stableID("execution",context.description.id,path,"accumulator"),element=stableID("execution",context.description.id,path,"element"),left=stableID("execution",context.description.id,path,"left"),right=stableID("execution",context.description.id,path,"right"),body=stableID("execution",context.description.id,path,"body");
    context.additions.push(graphEntity(accumulator,entity(accumulator,schema.iterationBinding,[[0x9f50,bytes(expression.accumulator)],[0x9f51,ref(ids.i64)]])),graphEntity(element,entity(element,schema.iterationBinding,[[0x9f50,bytes(expression.element)],[0x9f51,ref(ids.i64)]])),graphEntity(left,entity(left,schema.iterationRead,[[0x9f60,ref(accumulator)]])),graphEntity(right,entity(right,schema.iterationRead,[[0x9f60,ref(element)]])),graphEntity(body,entity(body,schema.integerAdd,[[0x9140,ref(left)],[0x9141,ref(right)],[0x9142,ref(ids.i64)]])),graphEntity(id,entity(id,schema.fold,[[0x9f70,ref(collectionRead)],[0x9f71,ref(initialRead)],[0x9f72,ref(accumulator)],[0x9f73,ref(element)],[0x9f74,ref(body)]])));return id;
  }
  if (expression.kind === "index") {
    const parameter = context.description.parameters.find((item) => item.name === expression.base);
    if(parameter?.type==="slice:i64"){
      if(result!=="i64"||!/^[A-Za-z_]/.test(expression.index))fail("lua.dynamic_index_type",expression.location);
      const base=parameterRead(expression.base,parameter.type,context,`${path}.collection`,expression.location),index=parameterRead(expression.index,"i64",context,`${path}.index`,expression.location);
      context.additions.push(graphEntity(id,entity(id,schema.dynamicIndexRead,[[0x9fa0,ref(base)],[0x9fa1,ref(index)]])));return id;
    }
    if (!parameter || !parameter.type.startsWith("array:")) fail("lua.index_collection_type", expression.location);
    const [, element, length] = parameter.type.split(":");
    if (result !== element || BigInt(expression.index) >= BigInt(length)) fail("lua.index_type_or_range", expression.location);
    const base = parameterRead(expression.base, parameter.type, context, `${path}.collection`, expression.location);
    const literal = stableID("execution", context.description.id, "expression", `${path}.index`, "integer-literal");
    context.additions.push(graphEntity(literal, entity(literal, schema.integerLiteral, [[0x9700, `uu ${expression.index}`], [0x9701, ref(ids.i64)]])));
    context.additions.push(graphEntity(id, entity(id, schema.indexRead, [[0x9f40, ref(base)], [0x9f41, ref(literal)]])));
    return id;
  }
  if(["collection_append","collection_update","slice_remove"].includes(expression.kind)){
    if(result!=="slice:i64")fail("lua.collection_mutation_result_type",expression.location);
    const parameter=context.description.parameters.find(item=>item.name===expression.base);if(!parameter||parameter.type!=="slice:i64")fail("lua.collection_mutation_receiver",expression.location);
    const base=parameterRead(expression.base,"slice:i64",context,`${path}.collection`,expression.location),fields=[];
    if(expression.kind==="collection_append"){const value=parameterRead(expression.value,"i64",context,`${path}.value`,expression.location);fields.push([0x9fb0,ref(base)],[0x9fb1,ref(value)]);context.additions.push(graphEntity(id,entity(id,schema.collectionAppend,fields)));return id;}
    const index=parameterRead(expression.index,"i64",context,`${path}.index`,expression.location);
    if(expression.kind==="slice_remove"){context.additions.push(graphEntity(id,entity(id,schema.sliceRemove,[[0xa0660,ref(base)],[0xa0661,ref(index)]])));return id;}
    const value=parameterRead(expression.value,"i64",context,`${path}.value`,expression.location);context.additions.push(graphEntity(id,entity(id,schema.collectionUpdate,[[0x9fc0,ref(base)],[0x9fc1,ref(index)],[0x9fc2,ref(value)]])));return id;
  }
  const mapParameter = context.description.parameters.find((item) => item.name === expression.base);
  if (!mapParameter || !mapParameter.type.startsWith("map:")) fail("lua.map_receiver_type", expression.location);
  const [, keyType, valueType] = mapParameter.type.split(":");
  const map = parameterRead(expression.base, mapParameter.type, context, `${path}.map`, expression.location);
  const key = parameterRead(expression.key, keyType, context, `${path}.key`, expression.location);
  if(expression.kind==="map_remove"){
    if(result!==mapParameter.type)fail("lua.map_remove_result_type",expression.location);
    context.additions.push(graphEntity(id,entity(id,schema.mapRemove,[[0xa0670,ref(map)],[0xa0671,ref(key)]])));return id;
  }
  if (expression.kind === "map_lookup") {
    const expected = `option:${valueType}`;
    if (result !== expected) fail("lua.map_lookup_option_type", expression.location);
    ensureType(expected, context.additions, context.records);
    context.additions.push(graphEntity(id, entity(id, schema.mapLookupOption, [[0xa0440, ref(map)], [0xa0441, ref(key)], [0xa0442, ref(typeID(expected))]])));
    return id;
  }
  if (expression.kind === "lookup_zero") {
    if (result !== valueType || expression.value || expression.descriptor !== valueType) fail("lua.map_lookup_type", expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.mapLookup, [[0xa0420, ref(map)], [0xa0421, ref(key)]])));
  } else {
    if (result !== mapParameter.type || !expression.value) fail("lua.map_update_type", expression.location);
    const value = parameterRead(expression.value, valueType, context, `${path}.value`, expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.mapUpdate, [[0xa0430, ref(map)], [0xa0431, ref(key)], [0xa0432, ref(value)]])));
  }
  return id;
}

function emitConstructor(expression, context, path) {
  const type = context.description.resultType;
  const isOption = type.startsWith("option:");
  const isResult = type.startsWith("result:");
  if ((expression.kind === "some" || expression.kind === "none") !== isOption || (expression.kind === "ok" || expression.kind === "err") !== isResult) fail("lua.constructor_result_type", expression.location);
  const id = stableID("execution", context.description.id, "expression", path, expression.kind);
  if (expression.kind === "none") {
    if (expression.arguments.length !== 0) fail("lua.constructor_arity", expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.optionNone, [[0xa0510, ref(typeID(type))]])));
    return id;
  }
  if (expression.arguments.length !== 1 || expression.arguments[0].kind !== "identifier") fail("lua.constructor_arity", expression.location);
  const argument = expression.arguments[0];
  const index = context.description.parameters.findIndex((parameter) => parameter.name === argument.name);
  if (index < 0) fail("lua.unknown_identifier", argument.location);
  const expected = expression.kind === "some" ? type.slice("option:".length) : splitResult(type)[expression.kind === "ok" ? 0 : 1];
  if (context.description.parameters[index].type !== expected) fail("lua.constructor_value_type", argument.location);
  const readID = stableID("execution", context.description.id, "expression", `${path}.value`, "parameter-read");
  context.additions.push(graphEntity(readID, entity(readID, schema.read, [[0x9130, ref(context.parameterIDs[index])]])));
  const constructorSchema = expression.kind === "some" ? schema.optionSome : expression.kind === "ok" ? schema.resultOk : schema.resultError;
  const fields = expression.kind === "some" ? [[0xa0520, ref(typeID(type))], [0xa0521, ref(readID)]] : expression.kind === "ok" ? [[0x9410, ref(typeID(type))], [0x9411, ref(readID)]] : [[0x9420, ref(typeID(type))], [0x9421, ref(readID)]];
  context.additions.push(graphEntity(id, entity(id, constructorSchema, fields)));
  return id;
}

function splitResult(type) {
  const parts = type.slice("result:".length).split(":");
  if(parts[0]==="transition"&&parts.length===5)return[parts.slice(0,4).join(":"),parts[4]];
  if (parts.length !== 2) fail("lua.nested_result_profile");
  return parts;
}
function typeID(type) {
  if (ids[type]) return ids[type];
  if(type==="function:i64:i64")return stableID("execution","type","function","i64","i64");
  if(type.startsWith("interface:"))return type.split(":")[1];
  if (type.startsWith("option:")) return stableID("execution", "type", "option", typeID(type.slice(7)));
  if (type.startsWith("result:")) { const [ok, error] = splitResult(type); return stableID("execution", "type", "result", typeID(ok), typeID(error)); }
  if (type.startsWith("array:")) { const [, element, length] = type.split(":"); return stableID("execution", "type", "fixed-array", element, length); }
  if (type.startsWith("slice:")) return stableID("execution", "type", "slice", type.slice(6));
  if (type.startsWith("map:")) { const [, key, value] = type.split(":"); return stableID("execution", "type", "map", key, value); }
  if (type.startsWith("record:")) return type.split(":")[1];
  if(type.startsWith("transition:")){const parts=type.split(":");return stableID("execution","type","state-transition",parts[1],typeID(parts[3]));}
  fail("lua.unknown_type");
}
function ensureType(type, additions, records) {
  const add = (id, schemaID, fields) => { if (!additions.some((item) => item.id === id)) additions.push(graphEntity(id, entity(id, schemaID, fields))); };
  if (type === "i64") return add(ids.i64, schema.i64Type, [[0x9100, "uu 64"], [0x9101, "tr"], [0x9102, "uu 0"]]);
  if (type === "bool") return add(ids.bool, schema.boolType, []);
  if (type === "text") return add(ids.text, schema.stringType, []);
  if (type === "bytes") return add(ids.bytes, schema.bytesType, []);
  if(type==="function:i64:i64"){ensureType("i64",additions,records);return add(typeID(type),schema.functionType,[[0xa0200,refs([ids.i64])],[0xa0201,ref(ids.i64)]]);}
  if(type.startsWith("interface:"))return;
  if (type.startsWith("option:")) { const value = type.slice(7); ensureType(value, additions, records); return add(typeID(type), schema.optionType, [[0xa0500, ref(typeID(value))]]); }
  if (type.startsWith("result:")) { const [ok, error] = splitResult(type); ensureType(ok, additions, records); ensureType(error, additions, records); return add(typeID(type), schema.resultType, [[0x9400, ref(typeID(ok))], [0x9401, ref(typeID(error))]]); }
  if (type.startsWith("array:")) { const [, element, length] = type.split(":"); ensureType(element, additions, records); return add(typeID(type), schema.fixedArrayType, [[0x9f20, ref(typeID(element))], [0x9f21, `uu ${length}`]]); }
  if (type.startsWith("slice:")) { const element = type.slice(6); ensureType(element, additions, records); return add(typeID(type), schema.sliceType, [[0x9f80, ref(typeID(element))]]); }
  if (type.startsWith("map:")) { const [, key, value] = type.split(":"); ensureType(key, additions, records); ensureType(value, additions, records); return add(typeID(type), schema.mapType, [[0xa0400, ref(typeID(key))], [0xa0401, ref(typeID(value))]]); }
  if(type.startsWith("transition:")){const parts=type.split(":");ensureType(`record:${parts[1]}:${parts[2]}`,additions,records);ensureType(parts[3],additions,records);return add(typeID(type),schema.transitionType,[[0xa0040,ref(parts[1])],[0xa0041,ref(typeID(parts[3]))]]);}
  if (type.startsWith("record:")) {
    const record = records.get(type.split(":").slice(2).join(":")); if (!record) fail("lua.unknown_record");
    const fieldIDs = record.fields.map((field, index) => { ensureType(field.type, additions, records); const id = stableID("execution", record.id, "field", String(index)); add(id, schema.recordField, [[0x9310, bytes(field.name)], [0x9311, ref(typeID(field.type))], [0x9312, `uu ${index}`]]); return id; });
    return add(record.id, schema.recordType, [[0x9300, bytes(record.name)], [0x9301, refs(fieldIDs)]]);
  }
  fail("lua.unknown_type");
}

function parseRecords(sources, packagePath) {
  const records = new Map();
  for (const { name: file, source } of sources) {
    const lines = source.replace(/\r\n?/g, "\n").split("\n");
    for (let index = 0; index < lines.length; index += 1) {
      const match = /^---@class\s+([A-Za-z_][A-Za-z0-9_]*)$/.exec(lines[index].trim());
      if (!match) continue;
      if (records.has(match[1])) fail("lua.duplicate_record", { file, line: index + 1, column: 1 });
      const record = { name: match[1], id: stableID("execution", "record", packagePath, match[1]), fields: [] };
      while (index + 1 < lines.length) {
        const field = /^---@field\s+([A-Za-z_][A-Za-z0-9_]*)\s+(\S+)$/.exec(lines[index + 1].trim());
        if (!field) break;
        record.fields.push({ name: field[1], sourceType: field[2], location: { file, line: index + 2, column: 1 } }); index += 1;
      }
      if (!record.fields.length || new Set(record.fields.map((field) => field.name)).size !== record.fields.length) fail("lua.invalid_record", { file, line: index + 1, column: 1 });
      records.set(record.name, record);
    }
  }
  for (const record of records.values()) record.fields = record.fields.map((field) => ({ ...field, type: annotationType(field.sourceType, field.location, records) }));
  return records;
}

const ids = {
  i64: stableID("execution", "type", "i64"),
  bool: stableID("execution", "type", "bool"),
  text: stableID("execution", "type", "string"),
  bytes: stableID("execution", "type", "bytes"),
};
function stableID(...parts) { const hash = crypto.createHash("sha256").update("seme.provider.identity.v1\0"); for (const part of parts) hash.update(part).update("\0"); return `80${hash.digest().subarray(0, 15).toString("hex")}`; }
function i64Word(value,location){let number;try{number=BigInt(value)}catch{fail("lua.invalid_i64_literal",location)}if(number<-(1n<<63n)||number>(1n<<63n)-1n)fail("lua.invalid_i64_literal",location);return BigInt.asUintN(64,number).toString();}
function graphEntity(id, text) { return { id, text }; }
function bytes(value) { return `by ${Buffer.from(value, "utf8").toString("hex") || "-"}`; }
function ref(value) { return `rf ${value}`; }
function refs(values) { return `li ${values.length}${values.map((value) => `\nrf ${value}`).join("")}`; }
function entity(id, type, fields) { fields.sort((a, b) => a[0] - b[0]); return `en ${id} ${type} 1 ${fields.length}\n${fields.map(([field, value]) => `fi ${field.toString(16).padStart(32, "0")} ${value}\n`).join("")}`; }
function compose(moduleG1, revision, additions) {
  if (typeof moduleG1 !== "string") fail("lua.requires_module");
  const entities = []; let current;
  for (const line of moduleG1.trim().split("\n")) { if (line.startsWith("en ")) { current = { id: line.split(/\s+/)[1], text: "" }; entities.push(current); } if (current) current.text += `${line}\n`; }
  const root = entities.find((item) => item.id === "00000000000000000000000000009000");
  if (Number(root?.text.match(/^en\s+\S+\s+\S+\s+(\d+)\s+/m)?.[1] ?? 0) >= 26) entities.length = 0;
  const unique = new Map(entities.map((item) => [item.id, item]));
  for (const item of additions) { const old = unique.get(item.id); if (old && old.text !== item.text) fail("lua.identity_collision"); unique.set(item.id, item); }
  const sorted = [...unique.values()].sort((a, b) => a.id.localeCompare(b.id));
  return `# Generated exact bounded Lua to Core Execution lift.\nve 1\nmo 00000000000000000000000000009000\nrv ${revision}\npc 0\nec ${sorted.length}\n${sorted.map((item) => `\n${item.text}`).join("")}`;
}
function fail(code, location) { throw new Error(`${code}${location ? `:${location.file}:${location.line}:${location.column}` : ""}`); }
