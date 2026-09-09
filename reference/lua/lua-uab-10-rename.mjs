import fs from "node:fs";

const [input, output, identity, before, after] = process.argv.slice(2);
if (!input || !output || !/^80[0-9a-f]{30}$/.test(identity) || !/^[A-Za-z_]\w*$/.test(before ?? "") || !/^[A-Za-z_]\w*$/.test(after ?? "")) throw new Error("usage: rename INPUT OUTPUT ID BEFORE AFTER");
const source = fs.readFileSync(input, "utf8");
const marker = `en ${identity} 00000000000000000000000000009011 `;
const start = source.indexOf(marker);
if (start < 0 || source.indexOf(marker, start + 1) >= 0) throw new Error("lua.identity_target");
const end = source.indexOf("\nen ", start + marker.length);
const entity = source.slice(start, end < 0 ? source.length : end);
const oldField = `fi 00000000000000000000000000009110 by ${Buffer.from(before).toString("hex")}`;
if (!entity.includes(oldField) || entity.indexOf(oldField) !== entity.lastIndexOf(oldField)) throw new Error("lua.identity_name_precondition");
const renamed = entity.replace(oldField, `fi 00000000000000000000000000009110 by ${Buffer.from(after).toString("hex")}`);
fs.writeFileSync(output, source.slice(0, start) + renamed + source.slice(start + entity.length));
