import fs from "node:fs";
const vectors = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
const i64 = value => ({kind:"i64", i64:String(value)}), bool = value => ({kind:"bool", bool:value});
console.log(JSON.stringify({
  valid: vectors.valid.map(item => ({name:item.name, arguments:[bool(item.scale),i64(item.amount),i64(item.value)], result:i64(item.result)})),
  malformed: [
    {name:"empty-request", arguments:[]},
    {name:"short-request", arguments:[bool(false),i64("3")]},
    {name:"long-request", arguments:[bool(false),i64("3"),i64("7"),i64("0")]},
    {name:"invalid-selector", arguments:[i64("0"),i64("3"),i64("7")]},
  ],
}));
