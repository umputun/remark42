module.exports = {
  extends: [
    'react-app',
    'preact',
    'plugin:jsx-a11y/recommended',
    'prettier',
    'prettier/@typescript-eslint',
    'prettier/babel',
    'prettier/prettier',
    'prettier/react',
  ],
  plugins: ['jsx-a11y', 'prettier'],
  rules: {
    'prettier/prettier': 'error',
  },
  overrides: [
    {
      files: ['*.ts?(x)'],
      parser: '@typescript-eslint/parser',
      rules: {
        'no-undef': 'off',
        'no-redeclare': 'off',
        'no-unused-vars': 'off',
      },
    },
    {
      files: ['*.d.ts'],
      rules: {
        '@typescript-eslint/no-unused-vars': 'off',
      },
    },
    {
      files: ['*.(spec|test).ts?(x)'],
      extends: ['react-app/jest'],
      rules: {
        'import/first': 'off',
      },
    },
  ],
};
