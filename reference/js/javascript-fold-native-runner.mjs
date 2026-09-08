const [moduleURL, valuesText = "", functionName = "Sum"] = process.argv.slice(2);
if (!moduleURL) throw new Error("usage: javascript-fold-native-runner MODULE CSV [FUNCTION]");
const values = valuesText === "" ? [] : valuesText.split(",").map((value) => {
  if (!/^-?\d+$/.test(value)) throw new Error("javascript_native.invalid_integer");
  return BigInt(value);
});
const module = await import(moduleURL);
const fn = module[functionName];
if (typeof fn !== "function") throw new Error("javascript_native.missing_function");
process.stdout.write(`${JSON.stringify({ result: String(fn(values)) })}\n`);
