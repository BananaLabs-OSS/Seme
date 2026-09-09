// Parser for the deliberately explicit Lua UAB-05 declaration surface. This
// is declaration parsing, not evaluation of arbitrary module-level Lua.
export function parseProtocolDeclarations(source, file = "input.lua") {
  const protocols = new Map(), implementations = new Map();
  const protocolPattern = /^local\s+([A-Za-z_]\w*)\s*=\s*Seme\.protocol\("([A-Za-z_]\w*)",\s*\{([^}]*)\}\)\s*$/gm;
  for (const match of source.matchAll(protocolPattern)) {
    const requirements = match[3].trim() ? match[3].split(",").map(item => {
      const value = /^\s*"([A-Za-z_]\w*)"\s*$/.exec(item);
      if (!value) fail("lua.protocol_requirement_syntax", file, source, match.index);
      return value[1];
    }) : [];
    if (!requirements.length || requirements.length > 32 || new Set(requirements).size !== requirements.length) fail("lua.protocol_requirements", file, source, match.index);
    if (protocols.has(match[1]) || [...protocols.values()].some(item => item.name === match[2])) fail("lua.duplicate_protocol", file, source, match.index);
    protocols.set(match[1], { binding: match[1], name: match[2], requirements });
  }
  const implementationPattern = /^local\s+([A-Za-z_]\w*)\s*=\s*Seme\.implementation\(([A-Za-z_]\w*),\s*"([A-Za-z_]\w*)",\s*\{([\s\S]*?)^\}\)\s*$/gm;
  for (const match of source.matchAll(implementationPattern)) {
    const protocol = protocols.get(match[2]);
    if (!protocol) fail("lua.implementation_protocol_scope", file, source, match.index);
    const methods = new Map();
    for (const entry of match[4].split(",")) {
      if (!entry.trim()) continue;
      const value = /^\s*([A-Za-z_]\w*)\s*=\s*([A-Za-z_]\w*)\s*$/.exec(entry);
      if (!value) fail("lua.implementation_method_syntax", file, source, match.index);
      if (methods.has(value[1])) fail("lua.duplicate_implementation_method", file, source, match.index);
      methods.set(value[1], value[2]);
    }
    if (methods.size !== protocol.requirements.length || protocol.requirements.some(name => !methods.has(name))) fail("lua.implementation_method_set", file, source, match.index);
    if (implementations.has(match[1])) fail("lua.duplicate_implementation", file, source, match.index);
    implementations.set(match[1], { binding: match[1], protocol: match[2], concreteKind: match[3], methods });
  }
  const declared = new Set([...protocols.keys(), ...implementations.keys()]);
  for (const name of declared) {
    const mutation = new RegExp(`^\\s*${name}\\s*(?:=|\\[|\\.)`, "m");
    if (mutation.test(source)) fail("lua.protocol_monkey_patch", file, source, source.search(mutation));
  }
  return { protocols, implementations };
}

export function stripProtocolDeclarations(source) {
  return source
    .replace(/^local\s+Seme\s*=\s*assert\(_G\.Seme,[^\n]*\)\s*$/gm, "")
    .replace(/^local\s+[A-Za-z_]\w*\s*=\s*Seme\.protocol\("[A-Za-z_]\w*",\s*\{[^}]*\}\)\s*$/gm, "")
    .replace(/^local\s+[A-Za-z_]\w*\s*=\s*Seme\.implementation\([A-Za-z_]\w*,\s*"[A-Za-z_]\w*",\s*\{[\s\S]*?^\}\)\s*$/gm, "");
}

function fail(code, file, source, index) {
  const line = source.slice(0, Math.max(0, index)).split("\n").length;
  throw new Error(`${code}:${file}:${line}:1`);
}
