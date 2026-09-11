import fs from "node:fs";
import {javascriptProjectManifest} from "./javascript-project-manifest.mjs";
const options=new Map();for(let i=2;i<process.argv.length;i+=2)options.set(process.argv[i],process.argv[i+1]);for(const key of ["--project","--graph","--out"])if(!options.has(key))throw new Error(`javascript_project_manifest.missing:${key}`);
const manifest=javascriptProjectManifest(options.get("--project"),JSON.parse(fs.readFileSync(options.get("--graph"),"utf8")));
fs.writeFileSync(options.get("--out"),`${JSON.stringify(manifest,null,2)}\n`,{flag:"wx",mode:0o600});
