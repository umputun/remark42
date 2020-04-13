module.exports = {
  hooks: {
    'pre-commit': 'lint-staged',
    'post-commit': 'git update-index --again',
    'pre-push': 'run-s check test',
  },
};
