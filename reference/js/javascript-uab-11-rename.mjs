import fs from "node:fs";

const [input, output] = process.argv.slice(2);
if (!input || !output) throw new Error("usage: rename INPUT OUTPUT");
const source = fs.readFileSync(input, "utf8");
const renamed = source.replace("export function Apply(", "export function Execute(");
if (renamed === source || renamed.includes("export function Apply(")) throw new Error("javascript_uab11.rename_target");
fs.writeFileSync(output, renamed);
