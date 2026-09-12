import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";

export const UPB12_AUTHORITY_HEADER="seme-upb12-source-free-authority-v1";
export const UPB12_REQUIRED=Object.freeze([
  "construction-v36.g1","execution-v36.seme","package-detail-v4.seme","package-v4.seme",
  "dependency-v1.seme","configuration-v3.seme","resource-v1.seme","durable-state-v1.seme",
  "source-presentation-v1.seme","ordered-transport-v1.seme","controlled-effects-v1.seme",
  "project-v12.seme","target-plan-v1.seme","project-v13.seme",
]);
const canonicalName=/^(?:[a-z0-9]+(?:-[a-z0-9]+)*\.(?:g1|seme|json)|blobs\/[0-9a-f]{64})$/;

export function loadUPB12Authority(root){
  root=strictDirectory(root);const manifestPath=path.join(root,"COMPLETE.sha256"),manifest=readRegular(manifestPath,1<<20);
  const lines=manifest.toString("utf8").split("\n");if(lines.at(-1)!=="")fail("manifest_newline");lines.pop();
  if(lines.shift()!==UPB12_AUTHORITY_HEADER)fail("manifest_header");
  const files=new Map(),seen=new Set();let prior="";
  for(const line of lines){const match=/^([^ ]+) ([0-9a-f]{64})$/.exec(line);if(!match)fail("manifest_line");const[,name,digest]=match;
    if(!canonicalName.test(name)||name.endsWith(".go")||name.endsWith(".js")||name.endsWith(".lua"))fail("native_source");
    if(name<=prior||seen.has(name))fail("manifest_order");prior=name;seen.add(name);
    const value=readRegular(path.join(root,...name.split("/")),64<<20);if(sha256(value)!==digest)fail("digest");
    if(name.startsWith("blobs/")&&name.slice(6)!==digest)fail("blob_identity");files.set(name,value);
  }
  for(const name of UPB12_REQUIRED)if(!files.has(name))fail(`required:${name}`);
  const entries=walk(root).filter(name=>name!=="COMPLETE.sha256");if(entries.length!==files.size||entries.some(name=>!files.has(name)))fail("undeclared_file");
  return Object.freeze({root,manifest:Buffer.from(manifest),files});
}

export function encodeUPB12Manifest(files){
  const names=[...files.keys()].sort();if(names.length!==files.size)fail("duplicate");
  for(const name of names)if(!canonicalName.test(name)||name.endsWith(".go")||name.endsWith(".js")||name.endsWith(".lua"))fail("native_source");
  return Buffer.from(`${UPB12_AUTHORITY_HEADER}\n${names.map(name=>`${name} ${sha256(files.get(name))}`).join("\n")}\n`);
}
function strictDirectory(value){if(!path.isAbsolute(value)||path.normalize(value)!==value)fail("root_path");const real=fs.realpathSync(value);if(real!==value||!fs.lstatSync(value).isDirectory())fail("root");return real;}
function readRegular(value,limit){const real=fs.realpathSync(value);if(real!==value)fail("symlink");const before=fs.lstatSync(value,{bigint:true});if(!before.isFile()||before.size<=0n||before.size>BigInt(limit))fail("regular");const data=fs.readFileSync(value);const after=fs.lstatSync(value,{bigint:true});if(before.dev!==after.dev||before.ino!==after.ino||before.size!==after.size||before.mtimeNs!==after.mtimeNs||BigInt(data.length)!==after.size)fail("changed");return data;}
function walk(root,current=""){const out=[];for(const entry of fs.readdirSync(path.join(root,current),{withFileTypes:true}).sort((a,b)=>a.name.localeCompare(b.name))){const relative=current?`${current}/${entry.name}`:entry.name,target=path.join(root,...relative.split("/"));if(entry.isSymbolicLink())fail("symlink");if(entry.isDirectory())out.push(...walk(root,relative));else if(entry.isFile())out.push(relative);else fail("special");}return out;}
function sha256(value){return crypto.createHash("sha256").update(value).digest("hex");}
function fail(code){throw new Error(`upb12_authority.${code}`);}
