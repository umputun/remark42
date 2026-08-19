const preactPreset = [
  '@babel/preset-react',
  {
    runtime: 'automatic',
    importSource: 'preact',
  },
];

// preset-env no longer injects core-js: babel 8 moved that to a separate plugin, and measuring
// it showed the previous useBuiltIns injection was worth about 50 bytes at these targets, while
// the replacement plugin adds 8.5 kB to embed.mjs regardless of targets
const plugins = ['module:fast-async'];
const removeTestId = './tasks/babel-plugin-remove-test-id.js';

module.exports = {
  targets: 'defaults, not IE 11, not samsung 12',
  presets: ['@babel/preset-env', preactPreset],
  plugins: [...plugins, removeTestId],
  env: {
    modern: {
      targets: { esmodules: true },
      presets: ['@babel/preset-env', preactPreset],
      plugins: [...plugins, removeTestId],
    },
    test: {
      targets: { node: 'current' },
      presets: ['@babel/preset-env', preactPreset],
      plugins,
    },
  },
};
