export function javascriptProjectManifest(projectIdentity, graph) {
  if(typeof projectIdentity!=="string"||!projectIdentity||!graph||!Array.isArray(graph.Packages)||graph.Packages.length===0)fail("input");
  const roots=graph.Packages.filter((pkg)=>pkg.Root);if(roots.length!==1)fail("root");
  const packages=graph.Packages.map((pkg)=>{
    const dependencies=[];const seen=new Set();
    for(const item of pkg.Imports??[]){const key=`${item.Requested}\0${item.Resolved}`;if(seen.has(key))continue;seen.add(key);dependencies.push({Name:item.Requested,Package:item.Resolved});}
    dependencies.sort((a,b)=>a.Name.localeCompare(b.Name)||a.Package.localeCompare(b.Package));
    const interfaces=(pkg.Members??[]).filter((member)=>member.Callable&&member.Visibility===2).map((member)=>({name:member.ExportName,function:member.Identity,parameters:[...(member.Parameters??[])],result:member.Results?.[0]})).sort((a,b)=>a.function.localeCompare(b.function));
    if(interfaces.some((item)=>!item.name||!item.result))fail(`interface:${pkg.Identity}`);
    return {name:pkg.Identity,interfaces,dependencies,effects:[...(pkg.Effects??[])]};
  }).sort((a,b)=>a.name.localeCompare(b.name));
  return {identity:projectIdentity,root_package:roots[0].Identity,packages};
}
function fail(code){throw new Error(`javascript_project_manifest.${code}`)}
