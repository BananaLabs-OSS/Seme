import fs from "node:fs";
const vectors=JSON.parse(fs.readFileSync(process.argv[2],"utf8")),i64=x=>({kind:"i64",i64:x}),result=x=>({kind:"result",variant:x.variant,payload:i64(x.payload)});
console.log(JSON.stringify({valid:vectors.valid.map(x=>({name:x.name,arguments:[i64(x.value)],result:result(x)})),malformed:[{name:"wrong-type",arguments:[{kind:"bool",bool:true}]},{name:"arity",arguments:[]}]}));
