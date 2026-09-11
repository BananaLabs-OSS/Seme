import fs from "node:fs";
import { reconcileJavaScriptProject } from "./javascript-project-reconcile.mjs";

const values=new Map();for(let index=2;index<process.argv.length;index+=2)values.set(process.argv[index],process.argv[index+1]);
const required=["--project","--out","--project-path","--files","--module","--entry","--target","--expected","--replacement","--revision","--native-runner"];
for(const name of required)if(!values.has(name))throw new Error(`javascript_reconcile.missing:${name}`);
reconcileJavaScriptProject({project:fs.realpathSync(values.get("--project")),destination:values.get("--out"),projectPath:values.get("--project-path"),files:values.get("--files").split(","),moduleG1:fs.realpathSync(values.get("--module")),entryName:values.get("--entry"),target:values.get("--target"),expected:values.get("--expected"),replacement:values.get("--replacement"),revision:Number(values.get("--revision")),nativeRunner:fs.realpathSync(values.get("--native-runner")),nativeArguments:values.has("--native-arguments")?values.get("--native-arguments").split(",").filter(Boolean):[]});
