import fs from "node:fs";

const [mode, profile, sourcePath, extra] = process.argv.slice(2);
const i64 = (value) => ({ kind: "i64", i64: String(value) }), bool = (value) => ({ kind: "bool", bool: value }), text = (value) => ({ kind: "text", text: value });
if (mode === "build") {
  const source = JSON.parse(fs.readFileSync(sourcePath, "utf8")); let valid;
  let malformed;
  if (profile === "scalar") { valid = source.valid.map((v) => ({ name:v.name, arguments:[i64(v.value),bool(v.enabled),text(v.label)], result:text(v.result) })); malformed=[{name:"i64-domain",arguments:[text("0"),bool(true),text("x")]},{name:"boolean-domain",arguments:[i64(0),i64(2),text("x")]},{name:"text-domain",arguments:[i64(0),bool(true),{kind:"bytes",bytes_hex:"c328"}]}]; }
  else if (profile === "collections") { valid = source.valid.map((v) => ({ name:v.name, arguments:[{kind:"record",fields:{value:i64(v.record)}},{kind:"array",items:v.array.map(i64)},{kind:"slice",items:v.slice.map(i64)},{kind:"map",value_type:"i64",entries:v.map.map(([key,value])=>({key:i64(key),value:i64(value)}))},i64(v.key)], result:i64(v.result) })); malformed=[{name:"missing-record-field",arguments:[{kind:"record",fields:{}},{kind:"array",items:[i64(1),i64(2)]},{kind:"slice",items:[]},{kind:"map",value_type:"i64",entries:[]},i64(0)]},{name:"short-fixed-array",arguments:[{kind:"record",fields:{value:i64(0)}},{kind:"array",items:[i64(1)]},{kind:"slice",items:[]},{kind:"map",value_type:"i64",entries:[]},i64(0)]},{name:"invalid-map-shape",arguments:[{kind:"record",fields:{value:i64(0)}},{kind:"array",items:[i64(1),i64(2)]},{kind:"slice",items:[],},{kind:"record",fields:{}},i64(0)]}]; }
  else if (profile === "composite") {
    const bytes = (hex) => ({kind:"bytes",bytes_hex:hex}), result = (variant,payload) => ({kind:"result",variant,payload}), option = (variant,payload) => ({kind:"option",variant,...(payload?{payload}:{})});
    valid=[{name:"none",arguments:[option("none")],result:bool(false)},{name:"ok",arguments:[option("some",result("ok",bytes("6f6b")))],result:bool(true)},{name:"wrong_bytes",arguments:[option("some",result("ok",bytes("6e6f")))],result:bool(false)},{name:"error",arguments:[option("some",result("error",text("bad")))],result:bool(true)},{name:"wrong_error",arguments:[option("some",result("error",text("no")))],result:bool(false)}]; malformed=[{name:"unknown-option",arguments:[option("unknown")]},{name:"missing-some-payload",arguments:[option("some")]},{name:"wrong-result-payload",arguments:[option("some",text("bad"))]}];
  } else throw new Error("canonical_vectors.profile");
  console.log(JSON.stringify({valid,malformed}));
} else if (mode === "normalize") {
  const observed = JSON.parse(fs.readFileSync(sourcePath,"utf8")), valid={};
  for(const [name,value] of Object.entries(observed.valid)){ if(value.kind==="i64")valid[name]=value.i64; else if(value.kind==="text")valid[name]=value.text??""; else if(value.kind==="bool")valid[name]=value.bool??false; else throw new Error("canonical_vectors.result_kind"); }
  console.log(JSON.stringify({valid,malformed:observed.malformed}));
} else throw new Error("canonical_vectors.mode");
