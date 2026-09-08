import { concatText, equalText, packTextFields, unpackTextFields } from "./string-abi-v2.mjs";

if (process.argv.length !== 4) throw new Error("usage: node string-abi-v2-runner.mjs concat|equal REQUEST_HEX");
const operation = process.argv[2];
const encoded = process.argv[3];
if (!/^(?:[0-9a-fA-F]{2})*$/.test(encoded)) throw new Error("string_abi_v2.invalid_hex");
const values = unpackTextFields(Uint8Array.from(Buffer.from(encoded, "hex")), 2);
let response;
if (operation === "concat") response = Buffer.from(new TextEncoder().encode(concatText(values[0], values[1]))).toString("hex");
else if (operation === "equal") response = equalText(values[0], values[1]) ? "01" : "00";
else throw new Error("string_abi_v2.invalid_operation");
console.log(JSON.stringify({ operation, response }));
