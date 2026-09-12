import fs from "node:fs";import path from "node:path";import {projectLuaModules} from "./lua-module-projector.mjs";
if(process.argv.length!==5)throw new Error("usage: lua-module-projector-cli CANONICAL GRAPH OUT");
const [,,canonical,graph,destination]=process.argv;
if(fs.existsSync(destination))throw new Error("lua_module_projector.output_exists");
const output=projectLuaModules(fs.readFileSync(canonical,"utf8"),JSON.parse(fs.readFileSync(graph,"utf8")));
const parent=path.dirname(destination),realParent=fs.realpathSync(parent);if(realParent!==parent)throw new Error("lua_module_projector.output_parent");
const stage=fs.mkdtempSync(path.join(parent,".lua-module-projector-"));let published=false;
try{for(const[name,source]of Object.entries(output)){const target=path.join(stage,name);if(!target.startsWith(`${stage}${path.sep}`))throw new Error("lua_module_projector.path");fs.mkdirSync(path.dirname(target),{recursive:true});fs.writeFileSync(target,source,{flag:"wx"});}fs.renameSync(stage,destination);published=true;}finally{if(!published)fs.rmSync(stage,{recursive:true,force:true});}
