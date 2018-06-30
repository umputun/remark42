module.exports = {
  presets: [
    [
      'env',
      {
        targets: {
          browsers: ['> 1%', 'android >= 4.4.4', 'ios >= 9', 'IE >= 11'],
        },
        useBuiltIns: true,
      },
    ],
  ],
  plugins: ['syntax-dynamic-import', 'transform-object-rest-spread', ['transform-react-jsx', { pragma: 'h' }]],
};
