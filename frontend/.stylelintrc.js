const path = require('path');
module.exports = {
  extends: ['stylelint-config-standard', 'stylelint-config-prettier'],
  plugins: ['stylelint-value-no-unknown-custom-properties', '@mavrin/stylelint-declaration-use-css-custom-properties'],
  rules: {
    'max-empty-lines': 1,
    'rule-empty-line-before': [
      'always-multi-line',
      {
        except: ['first-nested'],
        ignore: ['after-comment'],
      },
    ],
    'mavrin/stylelint-declaration-use-css-custom-properties': {
      cssDefinitions: ['color'],
      ignoreProperties: ['/^\\$/'],
      ignoreValues: ['/\\$/', 'transparent', '-webkit-focus-ring-color', 'currentColor'],
    },
    'csstools/value-no-unknown-custom-properties': [
      true,
      {
        importFrom: path.resolve(__dirname, './app/custom-properties.css'),
      },
    ],
  },
};
