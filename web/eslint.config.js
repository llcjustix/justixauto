const { defineConfig, globalIgnores } = require('eslint/config');
const js = require('@eslint/js');
const ts = require('typescript-eslint');
const hooks = require('eslint-plugin-react-hooks');
const refresh = require('eslint-plugin-react-refresh').default;
const globals = require('globals');
const prettier = require('eslint-config-prettier/flat');

// This config runs from web/ (npm run lint). The mock tooling (tools/) and the
// browser harness (e2e/) are not linted, as before they moved into web/.
module.exports = defineConfig([
  globalIgnores(['**/node_modules/**', '**/dist/**', '**/coverage/**', 'tools/**', 'e2e/**']),
  {
    files: ['**/*.{js,cjs,mjs,ts,tsx,mts,cts}'],
    extends: [js.configs.recommended],
    rules: { complexity: ['error', 20] },
  },
  {
    files: ['**/*.{ts,tsx,mts,cts}'],
    extends: [ts.configs.recommended],
  },
  {
    files: ['{apps,packages}/*/src/**/*.{ts,tsx}'],
    languageOptions: { globals: globals.browser },
    plugins: { 'react-hooks': hooks },
    rules: hooks.configs.recommended.rules,
  },
  {
    files: ['apps/*/src/**/*.tsx'],
    plugins: { 'react-refresh': refresh },
    rules: {
      'react-refresh/only-export-components': ['error', { allowConstantExport: true }],
    },
  },
  {
    files: ['**/*.config.{js,cjs,mjs,ts,mts,cts}', 'vitest.workspace.ts'],
    languageOptions: { globals: globals.node },
  },
  // Formatting belongs to Prettier; must stay last.
  prettier,
]);
