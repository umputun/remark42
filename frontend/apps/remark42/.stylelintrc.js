const { CUSTOM_PROPERTIES_PATH } = require('./webpack.config');

module.exports = {
  extends: ['stylelint-config-standard'],
  plugins: ['stylelint-value-no-unknown-custom-properties', 'stylelint-declaration-strict-value'],
  // lets property-layout-mappings/value-keyword-layout-mappings autofix a physical property to
  // the correct logical one (stylelint --fix, already run on commit by lint-staged): every logical
  // property in this codebase is authored as if ltr were the base direction, with postcss-preset-env
  // generating the [dir=rtl] variant at build time, so that is the direction stylelint normalizes from
  languageOptions: {
    directionality: { inline: 'left-to-right', block: 'top-to-bottom' },
  },
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
    // only the properties/keywords whose logical form is inline-start/inline-end (i.e. actually
    // direction-dependent under ltr vs rtl) are checked; everything excluded below maps to a
    // block-axis or size concept (block-start/end, inline-size/block-size, overflow-inline/block,
    // resize inline/block) that only differs under a vertical writing mode, which this project
    // does not support for any shipped locale — so it is out of scope, not merely noisy
    'property-layout-mappings': [
      'flow-relative',
      {
        ignoreProperties: [
          'margin-top',
          'margin-bottom',
          'padding-top',
          'padding-bottom',
          'border-top',
          'border-bottom',
          'border-top-width',
          'border-bottom-width',
          'border-top-style',
          'border-bottom-style',
          'border-top-color',
          'border-bottom-color',
          'top',
          'bottom',
          'width',
          'height',
          'min-width',
          'min-height',
          'max-width',
          'max-height',
          'scroll-margin-top',
          'scroll-margin-bottom',
          'scroll-padding-top',
          'scroll-padding-bottom',
          'overflow-x',
          'overflow-y',
          'overscroll-behavior-x',
          'overscroll-behavior-y',
          'contain-intrinsic-width',
          'contain-intrinsic-height',
        ],
      },
    ],
    'value-keyword-layout-mappings': [
      'flow-relative',
      {
        // resize's horizontal/vertical keywords map to inline/block (writing-mode axis), not
        // start/end (direction) — same reasoning as the block-axis properties above
        ignoreProperties: ['resize'],
      },
    ],
  },
  overrides: [
    {
      files: ['*.ejs', '**/*.ejs'],
      customSyntax: 'postcss-html',
      // standalone pages rather than the themeable widget surface, so literal colours are fine
      rules: {
        'scale-unlimited/declaration-strict-value': null,
        // these files are copied to production unprocessed, unlike the module CSS that
        // postcss-preset-env downlevels, so their media queries stay in the prefix form
        'media-feature-range-notation': 'prefix',
        // standalone dev/demo pages, deliberately left LTR-only regardless of widget locale — see
        // scale-unlimited/declaration-strict-value above for the same reasoning
        'property-layout-mappings': null,
        'value-keyword-layout-mappings': null,
      },
    },
  ],
};
