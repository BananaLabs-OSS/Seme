import { parse } from "acorn";
import { projectJavaScript } from "./javascript-projector.mjs";

export function projectJavaScriptModules(canonicalG1, graph) {
  if (!graph || !Array.isArray(graph.Packages) || graph.Packages.length < 2) fail("graph");
  const whole=projectJavaScript(canonicalG1);const comments=[];const ast=parse(whole,{ecmaVersion:2024,sourceType:"module",ranges:true,onComment:comments});
  const functions=new Map();const records=new Map();
  for(const item of ast.body){const declaration=item.type==="ExportNamedDeclaration"?item.declaration:item;if(declaration?.type!=="FunctionDeclaration")continue;let start=item.start;const prior=[...comments].reverse().find((comment)=>comment.type==="Block"&&comment.end<=start&&/^\s*$/.test(whole.slice(comment.end,start)));if(prior&&whole.slice(prior.start,prior.start+3)==="/**")start=prior.start;let source=whole.slice(start,item.end);if(item.type==="ExportNamedDeclaration"){const offset=item.start-start;source=`${source.slice(0,offset)}${source.slice(offset).replace(/^export\s+/,"")}`;}functions.set(declaration.id.name,source);}
  for(const comment of comments){const match=comment.value.match(/@typedef\s+\{Object\}\s+([A-Za-z_$][\w$]*)/);if(match){const fields=[...comment.value.matchAll(/@property\s+\{[^}]+\}\s+([A-Za-z_$][\w$]*)/g)].map((item)=>item[1]);const assignments=fields.map((field)=>`this.${field} = ${field};`).join(" ");records.set(match[1],`${whole.slice(comment.start,comment.end)}\nclass ${match[1]} { constructor(${fields.join(", ")}) { ${assignments} } }`);}}
  const byIdentity=new Map(graph.Packages.map((item)=>[item.Identity,item]));const output={};const used=new Set();
  for(const pkg of graph.Packages){if(!Array.isArray(pkg.Sources)||pkg.Sources.length!==1)fail(`sources:${pkg.Identity}`);const imports=[];for(const binding of pkg.Imports??[]){const target=byIdentity.get(binding.Resolved);if(!target)fail(`target:${binding.Resolved}`);const member=target.Members.find((item)=>item.ExportName===binding.Alias);if(!member)fail(`export:${binding.Alias}`);imports.push(`import { ${member.ExportName} } from ${JSON.stringify(binding.Requested)};`);}
    const bodies=[];for(const member of pkg.Members){let source=member.Callable?functions.get(member.Name):records.get(member.Name);if(!source)fail(`member:${member.Name}`);if(used.has(member.Name))fail(`duplicate:${member.Name}`);used.add(member.Name);if(member.Visibility===2){const token=member.Callable?"function ":"class ";const at=source.indexOf(token);if(at<0)fail(`declaration:${member.Name}`);source=`${source.slice(0,at)}export ${source.slice(at)}`;}bodies.push(source);}
    output[pkg.Sources[0].Path]=[...imports,bodies.join("\n\n")].filter(Boolean).join("\n\n")+"\n";
  }
  if(used.size!==functions.size+records.size)fail("coverage");return output;
}
function fail(code){throw new Error(`javascript_module_projector.${code}`)}
