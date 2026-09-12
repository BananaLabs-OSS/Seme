import fs from "node:fs";
import {loadUPB12Authority} from "./upb12-authority.mjs";

export function verifyUPB12Descriptor(descriptorPath,authorityRoot){
  const descriptor=JSON.parse(fs.readFileSync(descriptorPath,"utf8")),authority=loadUPB12Authority(authorityRoot);shape(descriptor);
  const parse=name=>JSON.parse(authority.files.get(name).toString("utf8"));
  const configuration=parse("configuration-selection-v1.json"),durable=parse("durable-selection-v1.json"),transport=parse("transport-selection-v1.json"),effects=parse("effects-selection-v1.json");
  const prefix=`${descriptor.identity}/`;const packages=new Set(descriptor.logical_packages.flatMap(role=>role.canonical_packages).map(name=>prefix+name));
  const references=[];visit([configuration,durable,transport,effects],(key,value)=>{if(["package","owner","owner_package","state_owner","port_owner"].includes(key)&&typeof value==="string"&&value.includes("/"))references.push(value)});
  if(references.length===0||references.some(value=>!value.startsWith(prefix)))fail("project_identity");
  const expectedPackages=new Set([prefix+"application",prefix+"configuration",prefix+"policy",prefix+"controlled",prefix+"state",prefix+"transport"]);if(packages.size!==expectedPackages.size||[...packages].some(value=>!expectedPackages.has(value)))fail("packages");
  if(!Array.isArray(configuration.fields)||configuration.fields.length!==3||configuration.fields.map(x=>x.type?.name).join(",")!=="bool,i64,string")fail("configuration");
  if(configuration.initializers?.map(x=>x.key).join(",")!=="configuration,policy,application")fail("initialization_order");
  if(durable.version!=="seme.durable-state-selection/v1"||durable.codec!=="seme.durable-state.canonical.v1"||durable.migration?.name!=="MigrateV1ToV2")fail("durable");
  if(transport.version!=="seme.ordered-transport-selection/v1"||transport.streams?.length!==1||transport.dispatch?.name!=="Dispatch"||transport.replay?.name!=="Replay")fail("transport");
  if(effects.version!=="seme.controlled-effects-selection/v1"||effects.clock?.injection_policy!=="explicit-replayable-input"||effects.random?.algorithm!=="state*48271+1"||effects.effect?.capability!==descriptor.controlled_effects.capability||effects.application?.dispatch?.name!==descriptor.entry.command||effects.application?.replay?.name!==descriptor.entry.replay)fail("effects");
  for(const expected of descriptor.resources)if(!authority.files.has(`blobs/${expected.sha256}`)||authority.files.get(`blobs/${expected.sha256}`).length!==expected.size)fail("resource_identity");
  return Object.freeze({identity:descriptor.identity,artifacts:authority.files.size,packages:[...packages].sort(),languages:[...descriptor.native_projections]});
}
function shape(d){if(d?.version!=="seme.upb12-project/v1"||d.identity!=="seme.upb12/service"||d.minimum_generated_observations<4096)fail("descriptor");if(d.native_projections?.join(",")!=="go,javascript,lua")fail("languages");if(d.logical_packages?.map(x=>x.role).join(",")!=="domain,durable,transport")fail("roles");if(d.semantic_edit?.kind!=="rename"||d.semantic_edit.expected!=="InitializePolicy"||d.semantic_edit.replacement!=="BuildPolicy"||d.semantic_edit.client_revision!==2)fail("edit");if(d.target?.name!=="wasm32-pulp-upb12-host-v1"||d.target?.rule_namespace!=="upb12")fail("target");}
function visit(value,fn){if(Array.isArray(value)){for(const item of value)visit(item,fn);return;}if(!value||typeof value!=="object")return;for(const[key,item]of Object.entries(value)){fn(key,item);visit(item,fn)}}
function fail(code){throw new Error(`upb12_descriptor.${code}`)}
