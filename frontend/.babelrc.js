module.exports = {
  presets: [
    [
      '@babel/preset-env',
      {
        bugfixes: true,
        loose: true,
      },
    ],
    [
      '@babel/preset-react',
      {
        pragma: 'h',
        pragmaFrag: 'Fragment',
      },
    ],
  ],
};
