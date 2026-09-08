const [moduleURL, firstText, secondText, functionName = "Observe"] = process.argv.slice(2);
if (!moduleURL || !["true", "false"].includes(firstText) || !["true", "false"].includes(secondText)) throw new Error("usage: runner MODULE true|false true|false [FUNCTION]");
const events = [];
const original = console.log;
console.log = (value) => events.push(value);
try {
  const module = await import(moduleURL);
  const result = module[functionName](firstText === "true", secondText === "true");
  process.stdout.write(`${JSON.stringify({ result, events })}\n`);
} finally {
  console.log = original;
}
