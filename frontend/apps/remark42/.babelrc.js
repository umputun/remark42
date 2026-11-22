const preactPreset = [
  '@babel/preset-react',
  {
    runtime: 'automatic',
    importSource: 'preact',
  },
];

// jest is all that reads this: the widget itself is compiled by the builder in frontend/build.
// The suites compile for the node running them
module.exports = {
  targets: { node: 'current' },
  presets: ['@babel/preset-env', preactPreset, '@babel/preset-typescript'],
};
