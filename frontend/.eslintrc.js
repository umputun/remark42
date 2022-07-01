module.exports = {
  root: true,
  extends: ['react-app', 'preact', 'plugin:jsx-a11y/recommended', 'prettier'],
  overrides: [
    {
      files: ['*.ts?(x)'],
      parser: '@typescript-eslint/parser',
      rules: {
        'no-undef': 'off',
        'no-redeclare': 'off',
        'no-unused-vars': 'off',
        '@typescript-eslint/no-explicit-any': 'error',
      },
    },
    {
      files: ['*.d.ts'],
      rules: {
        '@typescript-eslint/no-unused-vars': 'off',
        '@typescript-eslint/no-explicit-any': 'off',
      },
    },
    {
      files: ['./jest.config.ts'],
      rules: {
        'jest/no-jest-import': 'off',
      },
    },
    {
      files: ['*.@(test|spec).ts?(x)'],
      rules: {
        'import/first': 'off',
      },
    },
  ],
};
