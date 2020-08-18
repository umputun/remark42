module.exports = {
  hooks: {
    'pre-commit': 'lint-staged',
    'post-commit': 'git update-index --again',
  },
};
