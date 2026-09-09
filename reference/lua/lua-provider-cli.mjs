import fs from "node:fs";
import { liftLua } from "./lua-provider.mjs";

const values = new Map();
const sources = [];
for (let index = 2; index < process.argv.length; index += 2) {
  const key = process.argv[index];
  const value = process.argv[index + 1];
  if (!value) throw new Error(`lua.missing_value:${key}`);
  if (key === "--source") sources.push({ name: value, source: fs.readFileSync(value, "utf8") });
  else if (values.has(key)) throw new Error(`lua.duplicate_option:${key}`);
  else values.set(key, value);
}
for (const key of ["--module", "--package", "--revision", "--out", "--entry"]) {
  if (!values.has(key)) throw new Error(`lua.missing_option:${key}`);
}
fs.writeFileSync(values.get("--out"), liftLua({
  sources,
  moduleG1: fs.readFileSync(values.get("--module"), "utf8"),
  packagePath: values.get("--package"),
  revision: Number(values.get("--revision")),
  entryName: values.get("--entry"),
}));
