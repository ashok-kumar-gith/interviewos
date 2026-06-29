// Flat ESLint config (ESLint 9 / flat-config style) bridging the Next.js
// shareable config via FlatCompat. The `lint` npm script uses `next lint`
// (which reads .eslintrc.json) for Next 14 compatibility; this file lets
// tooling that prefers flat config (`eslint .`) resolve the same rules.
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({ baseDirectory: __dirname });

const config = [
  { ignores: [".next/**", "node_modules/**", "next-env.d.ts"] },
  ...compat.extends("next/core-web-vitals"),
];

export default config;
