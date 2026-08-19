import tsParser from '@typescript-eslint/parser';
import tsPlugin from '@typescript-eslint/eslint-plugin';
import preact from 'eslint-config-preact';
import jsxA11y from 'eslint-plugin-jsx-a11y';
import prettierRecommended from 'eslint-plugin-prettier/recommended';
import importPlugin from 'eslint-plugin-import';

export default [
  {
    ignores: ['node_modules/**', 'public/**', 'coverage/**', 'dist/**', '.turbo/**'],
  },

  ...preact,
  jsxA11y.flatConfigs.recommended,

  {
    plugins: { import: importPlugin },
    rules: {
      'import/first': 'error',
      'import/no-amd': 'error',
      'import/no-anonymous-default-export': 'warn',
      'import/no-webpack-loader-syntax': 'error',
    },
  },

  {
    files: ['**/*.ts', '**/*.tsx'],
    languageOptions: {
      parser: tsParser,
      parserOptions: { ecmaFeatures: { jsx: true } },
    },
    plugins: { '@typescript-eslint': tsPlugin },
    rules: {
      'no-undef': 'off',
      'no-redeclare': 'off',
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrors: 'none', ignoreRestSiblings: true },
      ],
      '@typescript-eslint/no-explicit-any': 'error',
    },
  },

  {
    files: ['**/*.d.ts'],
    rules: {
      '@typescript-eslint/no-unused-vars': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },

  {
    files: ['**/*.@(test|spec).ts?(x)'],
    rules: {
      'import/first': 'off',
    },
  },

  prettierRecommended,
];
