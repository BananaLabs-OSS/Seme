import fs from "node:fs";
const [input,output,mode]=process.argv.slice(2),source=fs.readFileSync(input,"utf8"),entities=[...source.matchAll(/^en\s+([0-9a-f]{32})\s+([0-9a-f]{32})\s+\d+\s+\d+\n(?:^(?!en\s)[^\n]*\n)*/gm)];
const bySchema=(suffix)=>entities.filter((item)=>item[2].endsWith(suffix));
let target,field;
if(mode==="witness-contract"){target=bySchema("a012")[0];field="a0121";}
else if(mode==="dynamic-requirement"){target=bySchema("a014")[0];field="a0141";}
else throw new Error("lua.uab05.unknown_forgery");
if(!target)throw new Error("lua.uab05.forgery_target");
const forged=target[0].replace(new RegExp(`^(fi [0-9a-f]*${field} rf )([0-9a-f]{32})$`,`m`),`$1${"00000000000000000000000000009010"}`);
if(forged===target[0])throw new Error("lua.uab05.forgery_field");
fs.writeFileSync(output,source.slice(0,target.index)+forged+source.slice(target.index+target[0].length));
