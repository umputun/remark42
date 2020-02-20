module.exports = {
  presets: [
    [
      '@babel/preset-env',
      {
        targets: {
          browsers: ['> 1%', 'android >= 4.4.4', 'ios >= 9', 'IE >= 11'],
        },
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
