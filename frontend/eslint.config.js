// @ts-check
const eslint = require('@eslint/js');
const tseslint = require('typescript-eslint');
const angular = require('@angular-eslint/eslint-plugin');
const template = require('@angular-eslint/eslint-plugin-template');
const templateParser = require('@angular-eslint/template-parser');

module.exports = tseslint.config(
  {
    // Exclude auto-generated files from linting. types.ts is produced by
    // `make types` (openapi-typescript) and must NOT be modified by hand —
    // any lint fixes added here would be erased on the next codegen run and
    // would corrupt the types-drift CI gate.
    ignores: ['src/app/api/types.ts'],
  },
  {
    files: ['**/*.ts'],
    extends: [
      eslint.configs.recommended,
      ...tseslint.configs.recommended,
    ],
    plugins: {
      '@angular-eslint': angular,
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
    },
  },
  {
    files: ['**/*.html'],
    plugins: {
      '@angular-eslint/template': template,
    },
    languageOptions: {
      parser: templateParser,
    },
    rules: {
      ...template.configs.recommended.rules,
    },
  }
);
