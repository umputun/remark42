module.exports = {
  './**/*.{ts,tsx,js,jsx}': ['eslint --fix --max-warnings=0', 'prettier --write'],
  './**/*.{scss,pcss,css}': ['prettier --write', 'stylelint'],
  './iframe.html': ['prettier --write', 'stylelint'],
};
