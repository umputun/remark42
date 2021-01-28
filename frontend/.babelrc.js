const getPresetEnv = (options) => ['@babel/preset-env', options];
const preactPreset = [
  '@babel/preset-react',
  {
    pragma: 'h',
    pragmaFrag: 'Fragment',
  },
];

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
    preactPreset,
  ],
  plugins,
  env: {
    modern: {
      presets: [getPresetEnv({ targets: { esmodules: true }, loose: true, bugfixes: true }), preactPreset],
      plugins,
    },
    test: {
      presets: [getPresetEnv({ targets: { node: 'current' } }), preactPreset],
      plugins,
    },
  },
};
