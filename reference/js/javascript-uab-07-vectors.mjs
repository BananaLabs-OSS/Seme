import fs from "node:fs";
const vectors=JSON.parse(fs.readFileSync(process.argv[2],"utf8")),i64=(x)=>({kind:"i64",i64:x}),counter=(x)=>({kind:"record",fields:{Value:i64(x)}});
console.log(JSON.stringify({valid:vectors.valid.map(x=>({name:x.name,arguments:[counter(x.value),i64(x.delta)],result:{kind:"transition",state:counter(x.state),result:i64(x.result)}})),malformed:[{name:"missing-record-field",arguments:[{kind:"record",fields:{}},i64("1")]},{name:"delta-is-bool",arguments:[counter("1"),{kind:"bool",bool:true}]},{name:"arity",arguments:[counter("1")]}]}));
