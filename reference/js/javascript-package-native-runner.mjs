import assert from "node:assert/strict";
import { Run } from "../../fixtures/javascript-uab-01/application.js";

assert.equal(Run(0n, 0n), 0n);
assert.equal(Run(7n, 5n), 12n);
assert.equal(Run(-7n, 5n), -2n);
