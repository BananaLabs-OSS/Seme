import fs from "node:fs";
import {buildLuaProjectGraph} from "./lua-project-graph.mjs";
const values=new Map();for(let i=2;i<process.argv.length;i+=2){if(values.has(process.argv[i]))throw new Error(`duplicate:${process.argv[i]}`);values.set(process.argv[i],process.argv[i+1]);}
for(const key of["--files","--snapshot","--canonical-g1","--project","--root-module","--out"])if(!values.has(key))throw new Error(`missing:${key}`);
const files=values.get("--files").split(",").map(path=>({path,source:fs.readFileSync(path,"utf8")}));
const graph=buildLuaProjectGraph({files,snapshot:JSON.parse(fs.readFileSync(values.get("--snapshot"),"utf8")),canonicalG1:fs.readFileSync(values.get("--canonical-g1"),"utf8"),projectPath:values.get("--project"),rootModule:values.get("--root-module")});
fs.writeFileSync(values.get("--out"),`${JSON.stringify(graph)}\n`,{flag:"wx"});
