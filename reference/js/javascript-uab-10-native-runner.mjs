import fs from "node:fs";
import { pathToFileURL } from "node:url";

const sourcePath = process.argv[2];
if (!sourcePath) throw new Error("usage: native-runner SOURCE");
const source = fs.readFileSync(sourcePath, "utf8");
const module = await import(`${pathToFileURL(sourcePath)}?uab10=${encodeURIComponent(source)}`);
const fn = module.Identity ?? module.Preserve;
if (typeof fn !== "function") throw new Error("javascript_uab10.entry_missing");
const inputs = [0n, 7n, -1n, 9223372036854775807n, -9223372036854775808n];
process.stdout.write(`${JSON.stringify(inputs.map((value) => ({ input: value.toString(), output: fn(value).toString() })))}\n`);
