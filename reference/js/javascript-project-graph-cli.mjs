import fs from "node:fs";
import { buildJavaScriptProjectGraph } from "./javascript-project-graph.mjs";

const options=new Map();for(let i=2;i<process.argv.length;i+=2)options.set(process.argv[i],process.argv[i+1]);
for(const name of ["--files","--snapshot","--canonical-g1","--project","--root-module","--out"])if(!options.has(name))throw new Error(`javascript_project_graph.missing:${name}`);
const files=options.get("--files").split(",").map((path)=>{const info=fs.lstatSync(path);if(!info.isFile()||info.isSymbolicLink())throw new Error(`javascript_project_graph.not_regular:${path}`);return {path,source:fs.readFileSync(path,"utf8")};});
const graph=buildJavaScriptProjectGraph({files,snapshot:JSON.parse(fs.readFileSync(options.get("--snapshot"),"utf8")),canonicalG1:fs.readFileSync(options.get("--canonical-g1"),"utf8"),projectPath:options.get("--project"),rootModule:options.get("--root-module"),identityEvidence:options.has("--identity-evidence")?JSON.parse(fs.readFileSync(options.get("--identity-evidence"),"utf8")):undefined});
fs.writeFileSync(options.get("--out"),`${JSON.stringify(graph,null,2)}\n`,{flag:"wx"});
