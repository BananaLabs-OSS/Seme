local source, entry = assert(arg[1]), assert(arg[2])
_G.Seme = {}
assert(loadfile(source))()
local fn = assert(_G[entry], "entry function is required")
assert(fn(false) == false)
assert(fn(true) == true)
io.write('{"false":false,"true":true}\n')
