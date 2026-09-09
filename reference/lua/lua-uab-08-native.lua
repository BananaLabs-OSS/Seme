local program,vectors=arg[1],arg[2];if not program or not vectors then error("usage: lua-uab-08-native PROGRAM VECTORS")end
local directory=assert(debug.getinfo(1,"S").source:match("^@(.*/)") );_G.Seme=dofile(directory.."seme-values.lua");dofile(program);local corpus,valid=vim.json.decode(table.concat(vim.fn.readfile(vectors),"\n")),{}
for _,item in ipairs(corpus.valid)do local observed=IncrementPositive(Seme.i64(item.value));local variant=Seme.result_is_ok(observed)and"ok"or"error";local payload=Seme.i64_decimal(Seme.result_value(observed));if variant~=item.variant or payload~=item.payload then error("lua.uab08:"..item.name)end;valid[item.name]={variant=variant,payload=payload}end
print(vim.json.encode({valid=valid}))
