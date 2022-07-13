module.exports = {
  printWidth: 120,
  useTabs: false,
  tabWidth: 2,
  semi: true,
  singleQuote: true,
  trailingComma: 'es5',
  bracketSpacing: true,
  overrides: [
    {
      files: ['*.html'],
      options: {
        trailingComma: 'none',
      },
    },
  ],
};
