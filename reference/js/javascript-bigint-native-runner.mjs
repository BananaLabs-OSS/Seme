const [moduleURL, originalText, replacementText, enabledText, functionName = "Choose"] = process.argv.slice(2);
if (!moduleURL || !["true", "false"].includes(enabledText)) {
  throw new Error("usage: javascript-bigint-native-runner MODULE ORIGINAL REPLACEMENT true|false [FUNCTION]");
}
const module = await import(moduleURL);
const fn = module[functionName];
if (typeof fn !== "function") throw new Error("javascript_native.missing_function");
process.stdout.write(`${JSON.stringify({ result: String(fn(BigInt(originalText), BigInt(replacementText), enabledText === "true")) })}\n`);
