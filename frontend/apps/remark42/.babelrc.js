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

// the e2e suite reports what it reaches in the widget the same way it does in the backend: the
// bundle counts what runs and the counters are read out of the page. Off unless asked for, so a
// published bundle never carries it.
const isCoverage = process.env.E2E_COVERAGE === '1';

// the widget document is served with `script-src 'self' 'unsafe-inline'`, and the default
// preamble reaches the global object through `new Function`, which that policy refuses outright:
// the bundle then throws before it renders anything
const istanbul = ['istanbul', { coverageGlobalScope: 'window', coverageGlobalScopeFunc: false }];

// core-js is not injected: the polyfill plugin costs 8.5 kB in embed.mjs and saves about
// 50 bytes at these targets
module.exports = {
  targets: isTest ? { node: 'current' } : 'defaults, not IE 11, not samsung 12',
  presets,
  plugins: isTest ? [] : [removeTestId, ...(isCoverage ? [istanbul] : [])],
};
