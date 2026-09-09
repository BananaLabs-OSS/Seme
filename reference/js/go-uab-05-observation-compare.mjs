import assert from "node:assert/strict";import fs from "node:fs";
const [vp,...paths]=process.argv.slice(2),vectors=JSON.parse(fs.readFileSync(vp)),valid=Object.fromEntries(vectors.valid.map(v=>[v.name,v.result])),rejected=Object.fromEntries(vectors.malformed.map(v=>[v.name,true]));
for(const path of paths){const x=JSON.parse(fs.readFileSync(path)),observed=Object.fromEntries(Object.entries(x.valid).map(([k,v])=>[k,v?.kind==="i64"?v.i64:String(v)]));assert.deepEqual(observed,valid,`${path}:valid`);if(x.rejected)assert.deepEqual(x.rejected,rejected,`${path}:rejected`)}
console.log("Go UAB-05 observations and named malformed linkage agree");
