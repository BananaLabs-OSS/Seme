local root,vectors,adapter=assert(arg[1]),assert(arg[2]),assert(arg[3]);package.path=root.."/?.lua;"..root.."/?/init.lua";local Seme=dofile(adapter);_G.Seme=Seme;local app=require("application");local f=assert(io.open(vectors,"r"));local data=vim.json.decode(f:read("*a"));f:close();local valid,rejected={},{}
local function decode(value) if value.kind~="i64" then error("native.wrong_type") end return Seme.i64_literal(value.i64) end
for _,item in ipairs(data.valid)do local args={};for i,value in ipairs(item.arguments)do args[i]=decode(value)end;local observed=Seme.i64_decimal(app.Run(unpack(args)));assert(observed==item.result.i64,item.name);valid[item.name]=observed end
for _,item in ipairs(data.malformed)do local ok=pcall(function()local args={};for i,value in ipairs(item.arguments)do args[i]=decode(value)end;return app.Run(unpack(args))end);assert(not ok,item.name);rejected[item.name]="rejected" end
io.write(vim.json.encode({valid=valid,rejected=rejected}),"\n")
