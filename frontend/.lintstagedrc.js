const configFiles = ['.lintstagedrc.js', '.stylelintrc.js', '.eslintrc.js'];
module.exports = {
  './**/*.{ts,tsx,js,jsx}': filenames => {
    const files = filenames
      .filter(file => {
        return !configFiles.some(configFile => file.endsWith(configFile));
      })
      .join(' ');
    return [`eslint  ${files} --max-warnings=0 --fix`, `git add ${files}`];
  },
  './**/*.{scss,pcss,css}': ['prettier --write', 'stylelint', 'git add'],
  './iframe.html': ['prettier --write', 'stylelint', 'git add'],
};
