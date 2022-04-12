const getPresetEnv = (options) => ['@babel/preset-env', options];

const plugins = ['module:fast-async'];

module.exports = {
  presets: [
    getPresetEnv({
      targets: 'defaults, not IE 11, not samsung 12',
      useBuiltIns: 'usage',
      corejs: 3,
      bugfixes: true,
      loose: true,
    }),
  ],
  plugins,
  env: {
    modern: {
      presets: [getPresetEnv({ targets: { esmodules: true }, loose: true, bugfixes: true })],
      plugins,
    },
    test: {
      presets: [getPresetEnv({ targets: { node: 'current' } })],
      plugins,
    },
  },
};
