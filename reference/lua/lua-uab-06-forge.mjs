import fs from "node:fs";
const[input,output,mode]=process.argv.slice(2),source=fs.readFileSync(input,"utf8");
const suffix=mode==="capture-type"?"a0301":mode==="update-capture"?"a0320":null;if(!suffix)throw Error("lua.uab06.forgery_mode");
const pattern=new RegExp(`^(fi [0-9a-f]*${suffix} rf )([0-9a-f]{32})$`,`m`),forged=source.replace(pattern,`$1${"00000000000000000000000000009020"}`);if(forged===source)throw Error("lua.uab06.forgery_field");fs.writeFileSync(output,forged);
