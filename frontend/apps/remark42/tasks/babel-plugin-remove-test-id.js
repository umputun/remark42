/**
 * Strips data-testid attributes so they do not ship to production.
 *
 * Replaces babel-plugin-jsx-remove-data-test-id, which is unmaintained and calls the
 * t.jSXOpeningElement builder that babel 8 renamed.
 */
module.exports = function removeTestId() {
  return {
    name: 'remove-test-id',
    visitor: {
      JSXAttribute(path) {
        const { name } = path.node.name;
        if (name === 'data-testid') {
          path.remove();
        }
      },
    },
  };
};
