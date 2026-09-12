import {editUPB12Native} from "./upb12-native-edit.mjs";
const values=new Map();for(let index=2;index<process.argv.length;index+=2)values.set(process.argv[index],process.argv[index+1]);
for(const name of ["--project","--graph","--language","--target","--expected","--replacement","--out"])if(!values.has(name))throw new Error(`upb12_native_edit.missing:${name}`);
editUPB12Native({projectRoot:values.get("--project"),graphPath:values.get("--graph"),language:values.get("--language"),target:values.get("--target"),expected:values.get("--expected"),replacement:values.get("--replacement"),destination:values.get("--out")});
