module.exports = {
  './**/*.{ts,tsx,js,jsx}': ['eslint --fix --max-warnings=0', 'prettier --write', 'git add'],
  './**/*.{scss,pcss,css}': ['prettier --write', 'stylelint', 'git add'],
  './iframe.html': ['prettier --write', 'stylelint', 'git add'],
};
