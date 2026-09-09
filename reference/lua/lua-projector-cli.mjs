import fs from "node:fs";
import { projectLua } from "./lua-projector.mjs";

if (process.argv.length !== 4) throw new Error("usage: node lua-projector-cli.mjs PROGRAM.g1 OUTPUT.lua");
fs.writeFileSync(process.argv[3], projectLua(fs.readFileSync(process.argv[2], "utf8")));
