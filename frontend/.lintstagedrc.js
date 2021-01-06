module.exports = {
  './**/*.{ts,tsx,js,jsx}': ['eslint --fix --max-warnings=0', 'prettier --write'],
  './**/*.css': ['prettier --write', 'stylelint'],
  './templates/**.html': ['prettier --write', 'stylelint'],
};
