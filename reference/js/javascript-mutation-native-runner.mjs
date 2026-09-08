const [moduleURL, value, suffix, enabledText, functionName = "AppendOnce"] = process.argv.slice(2);
if (!moduleURL || !["true", "false"].includes(enabledText)) {
  throw new Error("usage: javascript-mutation-native-runner MODULE VALUE SUFFIX true|false [FUNCTION]");
}
const module = await import(moduleURL);
const fn = module[functionName];
if (typeof fn !== "function") throw new Error("javascript_native.missing_function");
process.stdout.write(`${JSON.stringify({ result: fn(value, suffix, enabledText === "true") })}\n`);
