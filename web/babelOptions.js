module.exports = {
  presets: [
    ['env', {
      targets: ['> 1%', 'android >= 4.4.4', 'ios >= 9'],
      useBuiltIns: true,
    }],
  ],
  plugins: ['transform-object-rest-spread', ['transform-react-jsx', { 'pragma': 'h' }]],
};
