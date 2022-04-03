const { CUSTOM_PROPERTIES_PATH } = require('./webpack.config');

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
    'comment-empty-line-before': [
      'always',
      { except: ['first-nested'], ignore: ['after-comment', 'stylelint-commands'] },
    ],
    'value-keyword-case': ['lower', { ignoreProperties: ['composes'], camelCaseSvgKeywords: true }],
    'selector-pseudo-class-no-unknown': [true, { ignorePseudoClasses: ['global'] }],
    'property-no-unknown': [true, { ignoreProperties: ['composes'] }],
    'mavrin/stylelint-declaration-use-css-custom-properties': {
      cssDefinitions: ['color'],
      ignoreProperties: ['/^\\$/'],
      ignoreValues: ['/\\$/', 'transparent', '-webkit-focus-ring-color', 'currentColor'],
    },
    'csstools/value-no-unknown-custom-properties': [
      true,
      {
        importFrom: CUSTOM_PROPERTIES_PATH,
      },
    ],
    'selector-class-pattern': null,
    'color-function-notation': null,
    'shorthand-property-no-redundant-values': null,
    'alpha-value-notation': null,
    'declaration-block-no-redundant-longhand-properties': null,
  },
  overrides: [
    {
      files: ['*.html', '**/*.html', '*.ejs', '**/*.ejs'],
      customSyntax: 'postcss-html',
    },
  ],
};
