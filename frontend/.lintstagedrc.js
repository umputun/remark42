module.exports = {
  './**/*.{ts,tsx,js,jsx}': ['prettier --write', '"eslint --max-warnings=0 "**/*.{ts?(x),js}"'],
  './**/*.css': ['prettier --write', 'stylelint'],
  './templates/**.html': ['prettier --write', 'stylelint'],
};
