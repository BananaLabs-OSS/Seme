import fs from "node:fs";

const schema={program:"00000000000000000000000000009015",function:"00000000000000000000000000009011",parameter:"00000000000000000000000000009012",effect:"00000000000000000000000000000015"};

// emitLuaProjectManifest derives the Package-v1 input from canonical meaning.
// Source discovery and Lua syntax remain outside the neutral Project contract.
export function emitLuaProjectManifest(canonicalG1,identity){
  if(typeof canonicalG1!=="string"||typeof identity!=="string"||identity.length===0)fail("options");
  const graph=parse(canonicalG1),programs=[...graph.values()].filter((item)=>item.schema===schema.program);
  if(programs.length!==1)fail("program_cardinality");
  const entry=reference(field(programs[0],"00000000000000000000000000009151")),fn=required(graph,entry,schema.function);
  const name=text(field(fn,"00000000000000000000000000009110"));
  const parameters=references(field(fn,"00000000000000000000000000009111")).map((id)=>reference(field(required(graph,id,schema.parameter),"00000000000000000000000000009121")));
  const result=reference(field(fn,"00000000000000000000000000009112"));
  const effects=[...graph.values()].filter((item)=>item.schema===schema.effect).map((item)=>item.id).sort();
  return {identity,root_package:identity,packages:[{name:identity,interfaces:[{name,function:entry,parameters,result}],dependencies:[],effects}]};
}

function parse(source){const graph=new Map(),lines=source.replace(/\r\n?/g,"\n").split("\n");for(let index=0;index<lines.length;){if(!lines[index].startsWith("en ")){index++;continue;}const header=lines[index].split(/\s+/);if(header.length!==5||!/^[0-9a-f]{32}$/.test(header[1])||!/^[0-9a-f]{32}$/.test(header[2]))fail("entity");const item={id:header[1],schema:header[2],fields:new Map()},count=Number(header[4]),schemaDeclaration=item.id.startsWith("00");if(!Number.isSafeInteger(count)||count<0)fail("entity");index++;for(let seen=0;seen<count;seen++,index++){const match=/^fi\s+([0-9a-f]{32})\s+(by|rf|li|uu|tr|fa|rc)(?:\s+(.+))?$/.exec(lines[index]);if(!match)fail("field");if(match[2]==="li"){const length=Number(match[3]),values=[];if(!Number.isSafeInteger(length)||length<0)fail("list");for(let offset=0;offset<length;offset++){index++;const member=/^rf\s+([0-9a-f]{32})$/.exec(lines[index]);if(!member&&!schemaDeclaration)fail("list");if(member)values.push(member[1]);}item.fields.set(match[1],{kind:"li",value:values});}else item.fields.set(match[1],{kind:match[2],value:match[3]??""});}if(!schemaDeclaration){if(graph.has(item.id))fail("duplicate");graph.set(item.id,item);}}return graph;}
function required(graph,id,want){const item=graph.get(id);if(!item||item.schema!==want)fail("reference");return item;}
function field(item,id){const value=item.fields.get(id);if(!value)fail("missing_field");return value;}
function reference(value){if(value.kind!=="rf")fail("reference_value");return value.value;}
function references(value){if(value.kind!=="li")fail("list_value");return value.value;}
function text(value){if(value.kind!=="by"||value.value==="-")fail("text");try{return new TextDecoder("utf-8",{fatal:true}).decode(Buffer.from(value.value,"hex"));}catch{fail("text");}}
function fail(code){throw new Error(`lua_project_manifest.${code}`);}

if(import.meta.url===`file://${process.argv[1]}`){if(process.argv.length!==5)fail("usage");const value=emitLuaProjectManifest(fs.readFileSync(process.argv[2],"utf8"),process.argv[3]);fs.writeFileSync(process.argv[4],`${JSON.stringify(value,null,2)}\n`,{flag:"wx"});}
