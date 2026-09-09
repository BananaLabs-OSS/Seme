import fs from "node:fs";
const vectors=JSON.parse(fs.readFileSync(process.argv[2],"utf8"));
if(process.argv[3]==="--canonical"){
  console.log(JSON.stringify({valid:vectors.valid.map(x=>({name:x.name,arguments:[x.first,x.second],result:x.result,trace:x.trace}))}));
}else{
  for(const x of vectors.valid)console.log(`valid\t${x.name}\t${x.first?"01":"00"}${x.second?"01":"00"}\t${x.result?"01":"00"}\t${x.trace.join(",")}`);
  console.log("malformed\tempty\t-\t-\t-"); console.log(`malformed\tshort\t00\t-\t-`); console.log(`malformed\tlong\t000000\t-\t-`);
}
