import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import {encodeUPB12Manifest,loadUPB12Authority,UPB12_REQUIRED} from "./upb12-authority.mjs";

function fixture(){const root=fs.mkdtempSync(path.join(os.tmpdir(),"seme-upb12-authority-")),files=new Map(UPB12_REQUIRED.map((name,index)=>[name,Buffer.from(`canonical-${index}\n`)]));for(const[name,value]of files){const target=path.join(root,...name.split("/"));fs.mkdirSync(path.dirname(target),{recursive:true});fs.writeFileSync(target,value);}fs.writeFileSync(path.join(root,"COMPLETE.sha256"),encodeUPB12Manifest(files));return root;}
test("authenticates a closed source-free authority",()=>{const root=fixture();try{const got=loadUPB12Authority(root);assert.equal(got.files.size,UPB12_REQUIRED.length);assert.equal(got.root,root);}finally{fs.rmSync(root,{recursive:true,force:true});}});
test("rejects digest tamper, undeclared files, symlinks, and native source",()=>{for(const kind of ["digest","undeclared","symlink","source"]){const root=fixture();try{if(kind==="digest")fs.appendFileSync(path.join(root,UPB12_REQUIRED[0]),"x");if(kind==="undeclared")fs.writeFileSync(path.join(root,"extra.json"),"{}\n");if(kind==="symlink")fs.symlinkSync(path.join(root,UPB12_REQUIRED[0]),path.join(root,"alias.seme"));if(kind==="source"){const data=fs.readFileSync(path.join(root,"COMPLETE.sha256"),"utf8");fs.writeFileSync(path.join(root,"native.go"),"package x\n");fs.writeFileSync(path.join(root,"COMPLETE.sha256"),data+"native.go "+"0".repeat(64)+"\n");}assert.throws(()=>loadUPB12Authority(root),/upb12_authority\./);}finally{fs.rmSync(root,{recursive:true,force:true});}}});
