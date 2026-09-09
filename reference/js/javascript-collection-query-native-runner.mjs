const module = await import(process.argv[2]);
const fallback = BigInt(process.argv[3]);
const values = process.argv[4] === "" ? [] : process.argv[4].split(",").map(BigInt);
console.log(JSON.stringify({ result: module.LastOr(values, fallback).toString() }));
import { Seme } from "./seme-values.mjs";
globalThis.Seme = Seme;
