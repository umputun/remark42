/**
 * Strips data-testid attributes so they do not ship to production.
 *
 * `.babelrc.js` leaves this out of the test env, where the suites query by that attribute.
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
