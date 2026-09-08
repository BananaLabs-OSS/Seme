import fs from "node:fs";
import { projectJavaScript } from "./javascript-projector.mjs";

if (process.argv.length !== 4) throw new Error("usage: node javascript-projector-cli.mjs PROGRAM.g1 OUTPUT.mjs");
fs.writeFileSync(process.argv[3], projectJavaScript(fs.readFileSync(process.argv[2], "utf8")));
