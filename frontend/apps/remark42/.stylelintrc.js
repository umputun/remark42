const { CUSTOM_PROPERTIES_PATH } = require('./webpack.config');

module.exports = {
  extends: ['stylelint-config-standard'],
  plugins: ['stylelint-value-no-unknown-custom-properties', 'stylelint-declaration-strict-value'],
  rules: {
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
    'scale-unlimited/declaration-strict-value': [
      ['color'],
      {
        ignoreValues: ['transparent', 'inherit', 'currentColor', 'none', '-webkit-focus-ring-color'],
        ignoreFunctions: true,
        disableFix: true,
      },
    ],
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
    'selector-not-notation': null,
  },
  overrides: [
    {
      files: ['*.html', '**/*.html', '*.ejs', '**/*.ejs'],
      customSyntax: 'postcss-html',
      // standalone pages rather than the themeable widget surface, so literal colours are fine
      rules: {
        'scale-unlimited/declaration-strict-value': null,
      },
    },
  ],
};
