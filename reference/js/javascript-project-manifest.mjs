import {projectManifest} from "../project/project-manifest.mjs";
export function javascriptProjectManifest(projectIdentity,graph){try{return projectManifest(projectIdentity,graph);}catch(error){throw new Error(error.message.replace(/^project_manifest/,"javascript_project_manifest"));}}
