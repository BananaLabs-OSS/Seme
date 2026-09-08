import fs from "node:fs";
import { liftJavaScript } from "./javascript-provider.mjs";

const options = new Map();
for (let index = 2; index < process.argv.length; index += 2) options.set(process.argv[index], process.argv[index + 1]);
for (const name of ["--source", "--module", "--package", "--revision", "--out"]) {
  if (!options.has(name)) throw new Error(`javascript.missing_option:${name}`);
}
const canonical = liftJavaScript({
  source: fs.readFileSync(options.get("--source"), "utf8"),
  moduleG1: fs.readFileSync(options.get("--module"), "utf8"),
  packagePath: options.get("--package"),
  revision: Number(options.get("--revision")),
});
fs.writeFileSync(options.get("--out"), canonical);
