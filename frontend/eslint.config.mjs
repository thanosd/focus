import { defineConfig } from "eslint/config";
import nextCoreWebVitals from "eslint-config-next/core-web-vitals";
import nextTypescript from "eslint-config-next/typescript";

export default defineConfig([
  {
    extends: [...nextCoreWebVitals, ...nextTypescript],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector: "CallExpression[callee.name='fetch']",
          message:
            "Native fetch is banned. Please use apiClient from @/lib/api-client instead to ensure CSRF protection.",
        },
      ],
      // React Compiler advisory rules from eslint-config-next 16.2 flag
      // pre-existing patterns across the codebase. Tracked separately from
      // this security bump; revisit when migrating to React Compiler.
      "react-hooks/set-state-in-effect": "off",
      "react-hooks/preserve-manual-memoization": "off",
      "react-hooks/immutability": "off",
    },
  },
  {
    files: ["src/lib/api-client.ts"],
    rules: {
      "no-restricted-syntax": "off",
    },
  },
]);
