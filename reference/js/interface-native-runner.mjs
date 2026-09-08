import { pathToFileURL } from "node:url";

const sourcePath = process.argv[2];
if (!sourcePath) throw new Error("usage: node interface-native-runner.mjs <projected.mjs>");

const { Dispatch } = await import(pathToFileURL(sourcePath));
if (Dispatch(false, 3n, 4n) !== 7n) throw new Error("offset dispatch changed");
if (Dispatch(true, 3n, 4n) !== 12n) throw new Error("scale dispatch changed");
if (Dispatch(true, 2n, 9223372036854775807n) !== -2n) throw new Error("native JavaScript i64 overflow did not wrap");

console.log(JSON.stringify({ offset: "7", scale: "12", overflow: "wrapped", source: "native-javascript" }));
