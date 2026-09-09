import fs from "node:fs";
const vectors=JSON.parse(fs.readFileSync(process.argv[2],"utf8")), family=process.argv[3], i64=(x)=>({kind:"i64",i64:x});
console.log(JSON.stringify({valid:vectors[family].map(x=>({name:x.name,arguments:x.arguments.map(i64),result:i64(x.result)})),malformed:[{name:"arity",arguments:vectors[family][0].arguments.slice(1).map(i64)},{name:"wrong-type",arguments:[{kind:"bool",bool:true},...vectors[family][0].arguments.slice(1).map(i64)]}]}));
