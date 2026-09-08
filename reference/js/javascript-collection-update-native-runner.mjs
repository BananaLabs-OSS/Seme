const module = await import(process.argv[2]);
const values = process.argv[3] === "" ? [] : process.argv[3].split(",").map(BigInt);
const original = [...values];
const result = module.UpdateAndAppend(values, BigInt(process.argv[4]), BigInt(process.argv[5]), BigInt(process.argv[6]));
console.log(JSON.stringify({ result: result.map(String), input: values.map(String), unchanged: values.every((value, index) => value === original[index]) }));
