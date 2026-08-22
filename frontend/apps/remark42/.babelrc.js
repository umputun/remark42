const preactPreset = [
  '@babel/preset-react',
  {
    runtime: 'automatic',
    importSource: 'preact',
  },
];

const presets = ['@babel/preset-env', preactPreset, '@babel/preset-typescript'];
const removeTestId = './tasks/babel-plugin-remove-test-id.js';

// jest reads this same config and sets BABEL_ENV itself, and babel merges an `env` block over
// the root rather than replacing it, so the test branch is decided here rather than as an `env`:
// the suites query by data-testid and the stripper must not reach them.
const isTest = process.env.BABEL_ENV === 'test';

// core-js is not injected: the polyfill plugin costs 8.5 kB in embed.mjs and saves about
// 50 bytes at these targets
module.exports = {
  targets: isTest ? { node: 'current' } : 'defaults, not IE 11, not samsung 12',
  presets,
  plugins: isTest ? [] : [removeTestId],
};
