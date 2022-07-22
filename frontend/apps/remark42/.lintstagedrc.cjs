const eslint = 'eslint --fix';
const stylelint = 'stylelint --fix';
const prettier = 'prettier --write';

module.exports = {
  './**/*.{ts,tsx,js,jsx,cjs,mjs}': [eslint, prettier],
  './**/*.css': [stylelint, prettier],
  './templates/**.html': [stylelint, prettier],
};
