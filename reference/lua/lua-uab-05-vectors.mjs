import fs from "node:fs";
const vectors=JSON.parse(fs.readFileSync(process.argv[2],"utf8")),i64=(value)=>({kind:"i64",i64:value}),bool=(value)=>({kind:"bool",bool:value});
console.log(JSON.stringify({valid:vectors.valid.map((item)=>({name:item.name,arguments:[bool(item.use_double),i64(item.amount),i64(item.value)],result:i64(item.result)})),malformed:[{name:"selector-is-i64",arguments:[i64("0"),i64("3"),i64("7")]},{name:"amount-is-bool",arguments:[bool(false),bool(true),i64("7")]},{name:"missing-value",arguments:[bool(false),i64("3")]}]}));
