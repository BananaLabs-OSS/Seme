import fs from "node:fs";
import { liftJavaScriptPackage } from "./javascript-provider.mjs";

const options = new Map();
for (let index = 2; index < process.argv.length; index += 2) options.set(process.argv[index], process.argv[index + 1]);
for (const name of ["--files", "--module", "--package", "--revision", "--out"]) {
  if (!options.has(name)) throw new Error(`javascript_package.missing_option:${name}`);
}
const paths = options.get("--files").split(",");
const files = paths.map((path) => ({ path, source: fs.readFileSync(path, "utf8") }));
const canonical = liftJavaScriptPackage({
  files,
  moduleG1: fs.readFileSync(options.get("--module"), "utf8"),
  packagePath: options.get("--package"),
  revision: Number(options.get("--revision")),
  entryName: options.get("--entry"),
});
fs.writeFileSync(options.get("--out"), canonical);
