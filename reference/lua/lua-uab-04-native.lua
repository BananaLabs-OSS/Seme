Seme = dofile("reference/lua/seme-values.lua")
if not arg[1] or not arg[2] or not arg[3] then error("usage: runner PROGRAM VECTORS ENTRY") end
dofile(arg[1])
local handle=assert(io.open(arg[2],"rb"));local vectors=vim.json.decode(handle:read("*a"));handle:close()
local entry=arg[3];local function i(value)return Seme.i64(value)end
local function decode(value)if value.kind=="i64" then return i(value.i64) elseif value.kind=="slice" then local out={};for index,item in ipairs(value.items)do out[index]=decode(item)end;return Seme.slice(out) elseif value.kind=="map" then local entries={};for index,item in ipairs(value.entries or {})do entries[index]={key=decode(item.key),value=decode(item.value)}end;return Seme.map(value.value_type,entries) elseif value.kind=="option" and value.variant=="none" then return Seme.none() end;error("lua.uab04_vector_kind")end
local function describe(value)local parts={tostring(Seme.length(value))};for index=0,Seme.length(value)-1 do parts[#parts+1]=Seme.i64_decimal(Seme.index_zero(value,index))end;return table.concat(parts,",")end
local observed={}
for _,vector in ipairs(vectors.valid)do if vector.entry==entry then local arguments={};for index,value in ipairs(vector.arguments)do arguments[index]=decode(value)end;local returned=_G[entry](unpack(arguments));local result
  if vector.observe=="slice" then result=describe(returned)
  elseif vector.observe=="i64" then result=Seme.kind(returned)=="i64" and Seme.i64_decimal(returned) or tostring(returned)
  elseif vector.observe=="map-first-or-zero" then local key=vector.arguments[2] and decode(vector.arguments[2]) or i("5");result=Seme.i64_decimal(Seme.lookup_zero(returned,key)) end
  assert(result==vector.result);observed[vector.name]=result
end end
local malformed={}
for _,vector in ipairs(vectors.malformed)do if vector.entry==entry then local call=function()local arguments={};for index,value in ipairs(vector.arguments)do arguments[index]=decode(value)end;if vector.native_last_nil then arguments[#arguments]=nil end;return _G[entry](unpack(arguments))end;malformed[vector.name]=not pcall(call);assert(malformed[vector.name])end end
io.write(vim.json.encode({valid=observed,malformed=malformed}),"\n")
