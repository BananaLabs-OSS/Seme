import fs from "node:fs";
import { javascriptDeclarationIdentity } from "./javascript-provider.mjs";

const [packagePath, previousName, currentName, output] = process.argv.slice(2);
if (!packagePath || !previousName || !currentName || !output) throw new Error("usage: evidence PACKAGE PREVIOUS CURRENT OUTPUT");
fs.writeFileSync(output, `${JSON.stringify({
  version: 1,
  packagePath,
  renames: [{ previousName, currentName, identity: javascriptDeclarationIdentity(packagePath, previousName) }],
}, null, 2)}\n`);
