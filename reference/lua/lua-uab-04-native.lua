Seme = dofile("reference/lua/seme-values.lua")
if not arg[1] or not arg[2] or not arg[3] then error("usage: runner PROGRAM VECTORS ENTRY") end
dofile(arg[1])
local handle=assert(io.open(arg[2],"rb"));local vectors=vim.json.decode(handle:read("*a"));handle:close()
local entry=arg[3];local function i(value)return Seme.i64(value)end
local function slice(values)local out={};for index,value in ipairs(values)do out[index]=i(value)end;return Seme.slice(out)end
local function describe(value)local parts={tostring(Seme.length(value))};for index=0,Seme.length(value)-1 do parts[#parts+1]=Seme.i64_decimal(Seme.index_zero(value,index))end;return table.concat(parts,",")end
local observed={}
for _,vector in ipairs(vectors.valid)do if vector.entry==entry then local result
  if entry=="ConstructSlice" then result=describe(ConstructSlice(i("-1"),i("9223372036854775807")))
  elseif entry=="Append" then result=describe(Append(slice({"-1","1"}),i("2")))
  elseif entry=="Update" then result=describe(Update(slice({"1","2"}),i("0"),i("7")))
  elseif entry=="Remove" then result=describe(Remove(slice({"1","2"}),i("0")))
  elseif entry=="Length" then result=tostring(Length(slice({"1","2"})))
  elseif entry=="Index" then result=Seme.i64_decimal(Index(slice({"-1","2"}),i("0")))
  elseif entry=="Traverse" then result=Seme.i64_decimal(Traverse(slice({"9223372036854775807","1"}),i("0")))
  elseif entry=="EmptyMap" then result=Seme.i64_decimal(Seme.lookup_zero(EmptyMap(),i("5")))
  elseif entry=="Insert" then local map=Insert(Seme.empty_map("i64"),i("5"),i("9"));result=Seme.i64_decimal(Seme.lookup_zero(map,i("5")))
  elseif entry=="Lookup" then result=Seme.i64_decimal(Lookup(Seme.empty_map("i64"),i("5")))
  elseif entry=="RemoveMap" then local map=Seme.map_update(Seme.empty_map("i64"),i("5"),i("9"));result=Seme.i64_decimal(Seme.lookup_zero(RemoveMap(map,i("5")),i("5"))) end
  assert(result==vector.result);observed[vector.name]=result
end end
local malformed={}
local cases={
  ["index-oob"]=function() Index(slice({"1"}),i("1"))end,
  ["update-oob"]=function() Update(slice({"1"}),i("1"),i("2"))end,
  ["slice-remove-oob"]=function() Remove(slice({"1"}),i("1"))end,
  ["wrong-kind"]=function() Append(Seme.empty_map("i64"),i("1"))end,
  ["nil-map-delete"]=function() Insert(Seme.empty_map("i64"),i("1"),nil)end,
}
for _,vector in ipairs(vectors.malformed)do if vector.entry==entry then malformed[vector.name]=not pcall(cases[vector.name]);assert(malformed[vector.name])end end
io.write(vim.json.encode({valid=observed,malformed=malformed}),"\n")
