import { generatedSequences, canonicalValue } from "../js/javascript-uab-11-vectors.mjs";
process.stdout.write(JSON.stringify(canonicalValue(generatedSequences())) + "\n");
