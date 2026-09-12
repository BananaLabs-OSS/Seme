import fs from "node:fs";

if (process.argv.length !== 5) throw new Error("usage: upb12-runtime-corpus VECTORS REQUESTS EXPECTED");
const [vectorsPath, requestsPath, expectedPath] = process.argv.slice(2);
const vectors = {valid: []}, requests = [], expected = [];
for (let index = 0n; index < 4096n; index++) {
  let state = index * 7919n - 10000000n, delta = index * 104729n - 200000000n;
  if (index % 257n === 0n) [state, delta] = [9223372036854775807n, 1n];
  const arguments_ = [{kind: "record", fields: {Value: {kind: "i64", i64: String(state)}}}, {kind: "record", fields: {Delta: {kind: "i64", i64: String(delta)}}}];
  vectors.valid.push({name: `generated-${index}`, arguments: arguments_});
  requests.push(JSON.stringify({arguments: arguments_, capabilities: ["observability.log"]}));
  expected.push(JSON.stringify({value: {kind: "record", fields: {Value: {kind: "i64", i64: String(BigInt.asIntN(64, state + delta))}}}, effects: [{capability: "observability.log", value: true}]}));
}
fs.writeFileSync(vectorsPath, `${JSON.stringify(vectors)}\n`, {flag: "wx"});
fs.writeFileSync(requestsPath, `${requests.join("\n")}\n`, {flag: "wx"});
fs.writeFileSync(expectedPath, `${expected.join("\n")}\n`, {flag: "wx"});
