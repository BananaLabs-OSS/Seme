import fs from "node:fs";

const [mode, vectorsPath, observedPath, rejectionPath] = process.argv.slice(2);
if (!mode || !vectorsPath) throw new Error("usage: control-vector-adapter MODE VECTORS.json [OBSERVED]");
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const i64 = value => ({kind: "i64", i64: String(value)});
const bool = value => ({kind: "bool", bool: value});
const slice = values => ({kind:"slice",items:values.map(i64)});
const args = item => [i64(item.limit),bool(item.enabled),slice(item.values),i64(item.probe),bool(item.bypass)];
const request = item => {
  const bytes = Buffer.alloc(26+item.values.length*8);
  bytes.writeBigInt64LE(BigInt(item.limit));
  bytes[8] = item.enabled ? 1 : 0;
  bytes.writeUInt32LE(26,9);bytes.writeUInt32LE(item.values.length,13);
  bytes.writeBigInt64LE(BigInt(item.probe),17);bytes[25]=item.bypass?1:0;
  item.values.forEach((value,index)=>bytes.writeBigInt64LE(BigInt(value),26+index*8));
  return bytes.toString("hex");
};

if (mode === "canonical") {
  const malformed = vectors.malformed.map(item => {
    if(item.category==="checked-index")return{name:item.name,arguments:args(item)};
    if (item.category === "limit-text") return {name: item.name, arguments: [{kind: "text", text: "4"}, bool(true),slice(["99"]),i64(0),bool(false)]};
    if (item.category === "condition-i64") return {name: item.name, arguments: [i64(4), i64(1),slice(["99"]),i64(0),bool(false)]};
    if (item.category === "limit-overflow") return {name: item.name, arguments: [i64("9223372036854775808"), bool(true),slice(["99"]),i64(0),bool(false)]};
    throw new Error(`control-vector-adapter.category:${item.category}`);
  });
  console.log(JSON.stringify({
    valid: vectors.valid.map(item => ({name: item.name, arguments: args(item), result: i64(item.result)})),
    malformed,
  }));
} else if (mode === "requests") {
  for (const item of vectors.valid) console.log(request(item));
} else if(mode==="checked-requests"){
  for(const item of vectors.malformed.filter(item=>item.category==="checked-index"))console.log(`${item.name} ${request(item)}`);
} else if (mode === "normalize-pulp") {
  if (!observedPath) throw new Error("control-vector-adapter.observed");
  const lines = fs.readFileSync(observedPath, "utf8").trim().split("\n").filter(line => line.startsWith("{"));
  if (lines.length !== vectors.valid.length) throw new Error("control-vector-adapter.pulp_count");
  const valid = {};
  lines.forEach((line, index) => {
    const response = Buffer.from(JSON.parse(line).response, "hex");
    if (response.length !== 8) throw new Error("control-vector-adapter.pulp_response");
    valid[vectors.valid[index].name] = response.readBigInt64LE().toString();
  });
  if(!rejectionPath)throw new Error("control-vector-adapter.pulp_rejections");
  const rejected=fs.readFileSync(rejectionPath,"utf8").trim().split("\n").filter(Boolean);
  const expectedRejected=vectors.malformed.filter(item=>item.category==="checked-index").map(item=>item.name);
  if(JSON.stringify(rejected)!==JSON.stringify(expectedRejected))throw new Error("control-vector-adapter.pulp_rejection_names");
  console.log(JSON.stringify({valid, malformed: vectors.malformed.length}));
} else {
  throw new Error("control-vector-adapter.mode");
}
