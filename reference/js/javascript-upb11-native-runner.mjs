import { pathToFileURL } from "node:url";
import path from "node:path";
import { Seme } from "./seme-values.mjs";

const root=process.argv[2];if(!root)throw new Error("javascript_upb11_native.root");
globalThis.Seme=Seme;
const load=(name)=>import(pathToFileURL(path.join(root,name)).href+`?revision=${Date.now()}`);
const [application,configuration,policy]=await Promise.all([load("application.js"),load("configuration.js"),load("policy.js")]);
const candidates=["InitializePolicy","BuildPolicy"].filter((name)=>typeof policy[name]==="function");
if(candidates.length!==1)throw new Error("javascript_upb11_native.identity_binding");
const input={Enabled:configuration.DefaultEnabled(),Limit:configuration.DefaultLimit(),Namespace:configuration.DefaultNamespace()};
const checked=configuration.ValidateLimit(input.Limit);if(checked.tag!=="ok")throw new Error("validator");
const settings=configuration.InitializeConfig(input);if(settings.tag!=="ok")throw new Error("configuration");
const selected=policy[candidates[0]](settings.value);if(selected.tag!=="ok")throw new Error("policy");
const runtime=application.Assemble(settings.value,selected.value);if(runtime.tag!=="ok"||!runtime.value.Ready||runtime.value.Total!==16n||runtime.value.Namespace!=="seme")throw new Error("runtime");
process.stdout.write(`${JSON.stringify({lifecycle:["configuration","policy","application"],ready:runtime.value.Ready,total:runtime.value.Total.toString(),namespace:runtime.value.Namespace})}\n`);
