/* eslint-disable @typescript-eslint/camelcase */

module.exports = {
  parser: 'babel-eslint',
  extends: [
    'eslint:recommended',
    'plugin:jsx-a11y/recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:prettier/recommended',
  ],
  plugins: ['react', 'jsx-a11y', 'prettier'],
  overrides: [
    {
      files: ['*.ts', '*.tsx'],
      plugins: ['@typescript-eslint'],
      parser: '@typescript-eslint/parser',
      parserOptions: {
        project: './tsconfig.json',
        tsconfigRootDir: __dirname,
      },
      rules: {
        // disabling because typescipt uses it's own lint (see next rule)
        'no-unused-vars': 0,
        // allow Rust-like var starting with _underscore
        '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
        // disabling because it's bad practice to mark accessibility in react classes
        '@typescript-eslint/explicit-member-accessibility': 0,
        // doesn't work in real world
        '@typescript-eslint/no-non-null-assertion': 0,
        // disabling because store actions use WATCH_ME_IM_SPECIAL case
        '@typescript-eslint/class-name-casing': 0,
        // disabling because server response contains snake case
        '@typescript-eslint/camelcase': 0,
        // disabling because it's standard behaviour that function is hoisted to top
        '@typescript-eslint/no-use-before-define': 0,
        // well
        '@typescript-eslint/ban-ts-ignore': 0,
        // better to be explicit here maybe?
        '@typescript-eslint/no-inferrable-types': 0,
      },
    },
    {
      files: ['*.d.ts'],
      rules: {
        'no-var': 0,
      },
    },
    {
      files: ['*.test.ts', '*.test.tsx'],
      rules: {
        '@typescript-eslint/no-explicit-any': 0,
        '@typescript-eslint/no-object-literal-type-assertion': 0,
      },
      globals: {
        fail: true,
      },
    },
    {
      files: ['*.test.ts', '*.test.tsx', '*.test.js', '*.test.jsx'],
      rules: {
        'max-nested-callbacks': ['warn', { max: 10 }],
      },
    },
  ],
  env: {
    browser: true,
    node: true,
    es6: true,
    jest: true,
  },
  parserOptions: {
    ecmaVersion: 6,
    sourceType: 'module',
    ecmaFeatures: {
      modules: true,
      jsx: true,
    },
  },
  globals: {
    remark_config: true,
    __webpack_public_path__: true,
  },
  rules: {
    '@typescript-eslint/indent': 0,
    'react/jsx-uses-react': 2,
    'react/jsx-uses-vars': 2,
    'no-cond-assign': 1,
    'no-empty': ['error', { allowEmptyCatch: true }],
    'no-console': 1,
    camelcase: 0,
    'comma-style': 2,
    'max-nested-callbacks': [2, 3],
    'no-eval': 2,
    'no-implied-eval': 2,
    'no-new-func': 2,
    'guard-for-in': 2,
    eqeqeq: 2,
    'no-else-return': 2,
    'no-redeclare': 2,
    'no-dupe-keys': 2,
    radix: 2,
    strict: [2, 'never'],
    'no-shadow': 0,
    'callback-return': [1, ['callback', 'cb', 'next', 'done']],
    'no-delete-var': 2,
    'no-undef-init': 2,
    'no-shadow-restricted-names': 2,
    'handle-callback-err': 2,
    'no-lonely-if': 2,
    'constructor-super': 2,
    'no-this-before-super': 2,
    'no-dupe-class-members': 2,
    'no-const-assign': 2,
    'prefer-spread': 2,
    'prefer-const': 2,
    'no-useless-concat': 2,
    'no-var': 2,
    'object-shorthand': 2,
    'prefer-arrow-callback': 2,
    'prettier/prettier': 2,
    '@typescript-eslint/no-var-requires': 0,
    '@typescript-eslint/explicit-function-return-type': 0,
  },
};
