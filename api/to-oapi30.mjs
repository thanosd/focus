// Rewrites the bundled OpenAPI 3.1 spec into a 3.0-compatible copy for
// oapi-codegen, which does not understand `type: [X, "null"]` or
// `oneOf: [..., {type: "null"}]`. The frontend generator (openapi-typescript)
// reads the 3.1 original, so this file is a build artefact only.
//
// Usage: node to-oapi30.mjs openapi-bundled.yaml openapi-bundled-go.yaml
import { readFileSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const yaml = require("../frontend/node_modules/js-yaml");

const [, , input, output] = process.argv;
const doc = yaml.load(readFileSync(input, "utf8"));

function walk(node) {
  if (Array.isArray(node)) {
    node.forEach(walk);
    return;
  }
  if (!node || typeof node !== "object") return;
  if (Array.isArray(node.type)) {
    const types = node.type.filter((t) => t !== "null");
    if (types.length === 1) {
      node.type = types[0];
      if (node.type !== types.join()) node.nullable = true;
      if (node.type !== "null" && node.type.length) node.nullable = true;
    }
  }
  if (
    Array.isArray(node.oneOf) &&
    node.oneOf.some((s) => s && s.type === "null")
  ) {
    const rest = node.oneOf.filter((s) => !(s && s.type === "null"));
    delete node.oneOf;
    if (rest.length === 1) {
      Object.assign(node, rest[0]);
    } else {
      node.oneOf = rest;
    }
    node.nullable = true;
  }
  for (const value of Object.values(node)) walk(value);
}

walk(doc);
doc.openapi = "3.0.3";
writeFileSync(output, yaml.dump(doc, { lineWidth: -1, noRefs: true }));
