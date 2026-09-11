import { pathToFileURL } from "node:url";
import fs from "node:fs";
if(process.argv.length!==4)throw new Error("usage: native FIXTURE VECTORS");
const m=await import(pathToFileURL(`${process.argv[2]}/controlled.js`));
const vectors=JSON.parse(fs.readFileSync(process.argv[3],"utf8"));
for(const x of vectors.valid){
  const a=x.arguments,trace=[],original=console.log;
  console.log=value=>trace.push(value);
  let result;
  try{result=m.DispatchControlled(new m.ControlledState(BigInt(a[0].fields.Value.i64)),new m.ControlledCommand(BigInt(a[1].fields.Delta.i64)));}finally{console.log=original;}
  if(JSON.stringify(trace)!=="[true]")throw new Error("effect trace");
  process.stdout.write(`${JSON.stringify({value:{kind:"record",fields:{Value:{kind:"i64",i64:String(result.Value)}}},effects:[{capability:"observability.log",value:true}]})}\n`);
}
