const [moduleURL, firstText, secondText, thirdText, indexText, functionName = "Pick"] = process.argv.slice(2);
if (!moduleURL || [firstText, secondText, thirdText, indexText].some((value) => !/^-?\d+$/.test(value))) {
  throw new Error("usage: javascript-array-native-runner MODULE FIRST SECOND THIRD INDEX [FUNCTION]");
}
const module = await import(moduleURL);
const fn = module[functionName];
if (typeof fn !== "function") throw new Error("javascript_native.missing_function");
const result = fn(BigInt(firstText), BigInt(secondText), BigInt(thirdText), BigInt(indexText));
process.stdout.write(`${JSON.stringify({ result: String(result) })}\n`);
