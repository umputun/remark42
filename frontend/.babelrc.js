module.exports = {
  presets: [
    [
      '@babel/preset-env',
      {
        targets: {
          browsers: ['> 1%', 'android >= 4.4.4', 'ios >= 9', 'IE >= 11'],
        },
        useBuiltIns: 'usage',
        corejs: 3,
      },
    ],
    [
      '@babel/preset-react',
      {
        pragma: 'h',
        pragmaFrag: 'div',
      },
    ],
  ],
  plugins: ['@babel/plugin-syntax-dynamic-import', ['@babel/plugin-transform-react-jsx', { pragma: 'h' }]],
};
