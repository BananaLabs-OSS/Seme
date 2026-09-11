import fs from "node:fs";
import path from "node:path";
import { projectJavaScriptModules } from "./javascript-module-projector.mjs";
const [,,g1Path,graphPath,out]=process.argv;if(!g1Path||!graphPath||!out)throw new Error("usage: javascript-module-projector-cli <g1> <graph-json> <new-directory>");
if(fs.existsSync(out))throw new Error("javascript_module_projector.output_exists");const files=projectJavaScriptModules(fs.readFileSync(g1Path,"utf8"),JSON.parse(fs.readFileSync(graphPath,"utf8")));const stage=fs.mkdtempSync(`${path.dirname(out)}/.javascript-module-projector-`);let published=false;try{for(const [relative,source] of Object.entries(files)){const target=path.join(stage,relative);fs.mkdirSync(path.dirname(target),{recursive:true});fs.writeFileSync(target,source,{flag:"wx"});}fs.renameSync(stage,out);published=true;}finally{if(!published)fs.rmSync(stage,{recursive:true,force:true});}
