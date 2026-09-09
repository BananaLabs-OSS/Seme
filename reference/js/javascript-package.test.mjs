import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { liftJavaScript, liftJavaScriptPackage } from "./javascript-provider.mjs";
import { projectJavaScript } from "./javascript-projector.mjs";

const moduleG1 = fs.readFileSync(new URL("../../modules/execution/v30/module.g1", import.meta.url), "utf8");
const helper = {
  path: "math/sum.js",
  source: fs.readFileSync(new URL("../../fixtures/javascript-uab-01/math/sum.js", import.meta.url), "utf8").trim(),
};
const application = {
  path: "application.js",
  source: fs.readFileSync(new URL("../../fixtures/javascript-uab-01/application.js", import.meta.url), "utf8").trim(),
};

function lift(files) {
  return liftJavaScriptPackage({ files, moduleG1, packagePath: "example.test/multi-source", revision: 1, entryName: "Run" });
}

test("lifts a real cross-file call independently of input file order", () => {
  const forward = lift([application, helper]);
  const reverse = lift([helper, application]);
  assert.equal(reverse, forward);
  assert.equal((forward.match(/00000000000000000000000000009011/g) ?? []).length, 2);
  assert.equal((forward.match(/00000000000000000000000000009060/g) ?? []).length, 1);
});

test("executes the same files through the native ECMAScript module runtime", async () => {
  await import("./javascript-package-native-runner.mjs");
});

test("projects a multi-source package to executable native source and re-lifts exactly", () => {
  const canonical = lift([application, helper]);
  const projected = projectJavaScript(canonical);
  assert.match(projected, /function Sum\(left, right\)/);
  assert.match(projected, /export function Run\(base, delta\)/);
  new Function(projected.replace("export function", "function"));
  const relifted = liftJavaScript({ source: projected, moduleG1, packagePath: "example.test/multi-source", revision: 1, entryName: "Run" });
  assert.equal(relifted, canonical);
});

test("reports unsupported meaning at its source file", () => {
  const invalid = { ...helper, source: helper.source.replace("left + right", "left == right") };
  assert.throws(() => lift([application, invalid]), /javascript\.unsupported_expression:math\/sum\.js:2:\d+/);
});

test("rejects unresolved and host-package imports", () => {
  const missing = { ...application, source: application.source.replace("./math/sum.js", "./missing.js") };
  assert.throws(() => lift([missing, helper]), /javascript_package\.import_missing:application\.js:1:1/);
  const host = { ...application, source: application.source.replace("./math/sum.js", "node:fs") };
  assert.throws(() => lift([host, helper]), /javascript_package\.unsupported_import:application\.js:1:1/);
});

test("rejects duplicate and escaping package paths", () => {
  assert.throws(() => lift([helper, helper]), /javascript_package\.duplicate_file/);
  assert.throws(() => lift([{ ...helper, path: "../sum.js" }, application]), /javascript_package\.invalid_path/);
});
