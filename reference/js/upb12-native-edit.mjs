import fs from "node:fs";
import path from "node:path";

export function editUPB12Native({projectRoot, graphPath, language, target, expected, replacement, destination}) {
  if (!new Set(["go", "javascript", "lua"]).has(language) || !/^[0-9a-f]{32}$/.test(target) || !identifier(expected) || !identifier(replacement) || expected === replacement) fail("options");
  for (const value of [projectRoot, graphPath, destination]) if (!path.isAbsolute(value) || path.normalize(value) !== value) fail("path");
  const root = fs.realpathSync(projectRoot); if (root !== projectRoot || fs.existsSync(destination)) fail("path");
  const graph = JSON.parse(strictRead(graphPath));
  const matches = [];
  for (const pkg of graph.Packages ?? []) for (const member of pkg.Members ?? []) if (member.Identity === target) matches.push({pkg, member});
  if (matches.length !== 1 || matches[0].member.Name !== expected || matches[0].member.ExportName !== expected || !matches[0].member.Callable) fail("target");
  const source = matches[0].pkg.Sources;
  if (!Array.isArray(source) || source.length !== 1) fail("source");
  const relative = language === "go" ? `${matches[0].pkg.Identity.slice("seme.upb12/service/".length)}/seme_projected.go` : source[0].Path;
  if (language === "go" && !matches[0].pkg.Identity.startsWith("seme.upb12/service/")) fail("source");
  const sourcePath = path.join(root, ...relative.split("/"));
  let text = strictRead(sourcePath), edits;
  if (language === "go") edits = replaceExact(text, [`//seme:id ${target}\nfunc ${expected}(`], [`//seme:id ${target}\nfunc ${replacement}(`]);
  else if (language === "javascript") edits = replaceExact(text, [`export function ${expected}(`], [`export function ${replacement}(`]);
  else {
    if (!text.includes(`---@seme-id ${target}\n`)) fail("source_identity");
    const first = replaceExact(text, [`local function ${expected}(`], [`local function ${replacement}(`]);
    const second = replaceExact(first.text, [` ${expected} = ${expected} `], [` ${replacement} = ${replacement} `]);
    edits = {text: second.text, count: first.count + second.count};
  }
  const parent = fs.realpathSync(path.dirname(destination)); if (parent !== path.dirname(destination)) fail("destination_parent");
  const stage = fs.mkdtempSync(path.join(parent, ".seme-upb12-edit-")); let published = false;
  try {
    const candidate = path.join(stage, "project"); copyTree(root, candidate);
    fs.writeFileSync(path.join(candidate, ...relative.split("/")), edits.text);
    const report = {version: "seme.upb12-native-edit/v1", target, field: "00000000000000000000000000009110", expected, replacement, client_revision: 2};
    fs.writeFileSync(path.join(candidate, "SEME-EDIT.json"), `${JSON.stringify(report)}\n`, {flag: "wx"});
    fs.renameSync(candidate, destination); published = true; return report;
  } finally { if (!published) fs.rmSync(stage, {recursive: true, force: true}); else fs.rmdirSync(stage); }
}

function replaceExact(text, beforeParts, afterParts) { const before=beforeParts.join(""),after=afterParts.join("");const index=text.indexOf(before);if(index<0||text.indexOf(before,index+before.length)>=0)fail("occurrence");return{text:text.slice(0,index)+after+text.slice(index+before.length),count:1}; }
function strictRead(name){const real=fs.realpathSync(name);if(real!==name)fail("symlink");const stat=fs.lstatSync(name);if(!stat.isFile()||stat.size<=0||stat.size>64<<20)fail("regular");return fs.readFileSync(name,"utf8");}
function copyTree(from,to){fs.mkdirSync(to);for(const entry of fs.readdirSync(from,{withFileTypes:true})){if(entry.isSymbolicLink())fail("symlink");const source=path.join(from,entry.name),targetPath=path.join(to,entry.name);if(entry.isDirectory())copyTree(source,targetPath);else if(entry.isFile())fs.copyFileSync(source,targetPath,fs.constants.COPYFILE_EXCL);else fail("special");}}
function identifier(value){return typeof value==="string"&&/^[A-Za-z_][A-Za-z0-9_]*$/.test(value)}
function fail(code){throw new Error(`upb12_native_edit.${code}`)}
