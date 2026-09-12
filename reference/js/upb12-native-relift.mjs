import fs from "node:fs";
import path from "node:path";
import {loadUPB12Authority} from "./upb12-authority.mjs";
import {projectJavaScriptModules} from "./javascript-module-projector.mjs";
import {projectLuaModules} from "../lua/lua-module-projector.mjs";

export function reliftUPB12Native({authorityRoot,graphPath,projectRoot,language}){
  if(language!=="javascript"&&language!=="lua")fail("language");const authority=loadUPB12Authority(authorityRoot),canonical=authority.files.get("construction-v36.g1"),graph=JSON.parse(read(graphPath).toString("utf8"));
  const expected=language==="javascript"?projectJavaScriptModules(canonical.toString("utf8"),graph):projectLuaModules(canonical.toString("utf8"),graph),extension=language==="javascript"?".js":".lua",allowedDetached=new Set(language==="javascript"?["native-test.mjs"]:["native-test.lua","seme-values.lua"]);
  const found=[];walk(projectRoot,"",found);for(const name of found){if(name.endsWith(extension)&&!allowedDetached.has(name)&&!Object.hasOwn(expected,name))fail(`unexpected_source:${name}`)}
  for(const[name,value]of Object.entries(expected)){const target=path.join(projectRoot,...name.split("/"));if(read(target).toString("utf8")!==value)fail(`source_mismatch:${name}`)}
  return Buffer.from(canonical);
}
function read(name){const real=fs.realpathSync(name);if(real!==name)fail("symlink");const before=fs.lstatSync(name,{bigint:true});if(!before.isFile()||before.size<=0n||before.size>(64n<<20n))fail("regular");const value=fs.readFileSync(name),after=fs.lstatSync(name,{bigint:true});if(before.dev!==after.dev||before.ino!==after.ino||before.size!==after.size||before.mtimeNs!==after.mtimeNs||BigInt(value.length)!==after.size)fail("changed");return value;}
function walk(root,relative,out){const directory=relative?path.join(root,...relative.split("/")):root;for(const entry of fs.readdirSync(directory,{withFileTypes:true})){const name=relative?`${relative}/${entry.name}`:entry.name;if(entry.isSymbolicLink())fail("symlink");if(entry.isDirectory())walk(root,name,out);else if(entry.isFile())out.push(name);else fail("special")}}
function fail(code){throw new Error(`upb12_native_relift.${code}`)}
