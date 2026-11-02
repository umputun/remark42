import tsParser from '@typescript-eslint/parser';
import tsPlugin from '@typescript-eslint/eslint-plugin';
import preact from 'eslint-config-preact';
import jsxA11y from 'eslint-plugin-jsx-a11y';
import prettierRecommended from 'eslint-plugin-prettier/recommended';
import importPlugin from 'eslint-plugin-import';
import confusingBrowserGlobals from 'confusing-browser-globals';

const config = [
  {
    ignores: ['node_modules/**', 'public/**', 'coverage/**', 'dist/**'],
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
      '@typescript-eslint/no-unused-expressions': [
        'error',
        { allowShortCircuit: true, allowTernary: true, allowTaggedTemplates: true },
      ],
      '@typescript-eslint/no-use-before-define': ['error', { functions: false, classes: false, variables: false }],
      '@typescript-eslint/no-redeclare': 'error',
      '@typescript-eslint/no-array-constructor': 'error',
      '@typescript-eslint/no-useless-constructor': 'error',
      '@typescript-eslint/consistent-type-assertions': 'warn',
      // babel strips types without a type checker, so an unmarked type-only import keeps its
      // module in the bundle; marking them is what lets `verbatimModuleSyntax` hold
      '@typescript-eslint/consistent-type-imports': ['error', { disallowTypeAnnotations: false }],
      'no-array-constructor': 'off',
      'no-useless-constructor': 'off',
      'no-use-before-define': 'off',
      'no-unused-expressions': 'off',
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

  // project rules that none of the shared configs above turn on. last in the array on purpose:
  // eslint-config-prettier, which prettierRecommended pulls in, switches several of them off
  {
    rules: {
      'prefer-arrow-callback': 'error',
      'no-restricted-globals': ['error', ...confusingBrowserGlobals],
      'no-restricted-properties': [
        'error',
        { object: 'require', property: 'ensure', message: 'Use import() instead.' },
        { object: 'System', property: 'import', message: 'Use import() instead.' },
      ],
      'no-restricted-syntax': ['warn', 'WithStatement'],
      // type imports are a separate statement by design here, so a module legitimately
      // appears twice: once for values and once for types
      'no-duplicate-imports': ['error', { allowSeparateTypeImports: true }],
      'array-callback-return': 'warn',
      'no-array-constructor': 'warn',
      eqeqeq: ['warn', 'smart'],
      'no-eval': 'warn',
      'no-extend-native': 'warn',
      'no-implied-eval': 'warn',
      'no-inner-declarations': 'error',
      'no-loop-func': 'warn',
      'no-new-func': 'warn',
      'no-script-url': 'warn',
      'no-self-compare': 'warn',
      'no-sequences': 'warn',
      'no-template-curly-in-string': 'warn',
      'no-throw-literal': 'warn',
      'react/jsx-pascal-case': ['warn', { allowAllCaps: true, ignore: [] }],
      'react/no-danger-with-children': 'warn',
      'react/no-direct-mutation-state': 'warn',
      'react/no-typos': 'error',
      'react/style-prop-object': 'warn',
      'no-extra-bind': 'warn',
      'no-extra-label': 'warn',
      'no-label-var': 'warn',
      'no-labels': ['warn', { allowLoop: true, allowSwitch: false }],
      'no-lone-blocks': 'warn',
      'no-object-constructor': 'warn',
      'no-octal-escape': 'warn',
      'no-unused-expressions': ['error', { allowShortCircuit: true, allowTernary: true, allowTaggedTemplates: true }],
      'no-use-before-define': ['warn', { functions: false, classes: false, variables: false }],
      'default-case': ['warn', { commentPattern: '^no default$' }],
    },
  },

  // the base rules above are the typescript-eslint versions' business on TS files
  {
    files: ['**/*.ts', '**/*.tsx'],
    rules: {
      'no-unused-expressions': 'off',
      'no-use-before-define': 'off',
      'default-case': 'off',
      'no-array-constructor': 'off',
    },
  },
];

export default config;
