import fs from "node:fs";
import { readJavaScriptReconciliation } from "./javascript-project-reconcile.mjs";

const values=new Map();for(let index=2;index<process.argv.length;index+=2)values.set(process.argv[index],process.argv[index+1]);
for(const name of ["--project","--project-path","--files","--module","--entry","--native-runner"])if(!values.has(name))throw new Error(`javascript_reconcile_verify.missing:${name}`);
const result=readJavaScriptReconciliation({project:fs.realpathSync(values.get("--project")),projectPath:values.get("--project-path"),files:values.get("--files").split(","),structuredReferences:values.has("--structured-references")?values.get("--structured-references").split(",").filter(Boolean):[],moduleG1:fs.realpathSync(values.get("--module")),entryName:values.get("--entry"),nativeRunner:fs.realpathSync(values.get("--native-runner"))});
process.stdout.write(`${JSON.stringify(result.report,null,2)}\n`);
