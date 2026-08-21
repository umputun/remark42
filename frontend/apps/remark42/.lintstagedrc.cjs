const eslint = 'eslint --fix';
const stylelint = 'stylelint --fix';
const prettier = 'prettier --write';

module.exports = {
  // eslint --fix runs prettier through the prettier/prettier rule, so it is not repeated here
  './**/*.{ts,tsx,js,jsx,cjs,mjs}': [eslint],
  './**/*.css': [stylelint, prettier],
  './templates/**.html': [stylelint, prettier],
};
