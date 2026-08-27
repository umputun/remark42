const process = require('process');
const locales = require('./supportedLocales');
const { getLocalePath } = require('./getLocalePath');
const { keys } = require('./getTranslationKeys');

const errors = [];

locales.forEach((locale) => {
  const dict = require(getLocalePath({ locale }));
  const keysFromDict = Object.keys(dict);
  keysFromDict.forEach((key) => {
    if (!keys.includes(key)) {
      errors.push(
        `"${key}" key not found in "${locale}" locale dict. Please run "pnpm translation:generate" and commit changes.`
      );
    }
    return null;
  });
});
// the loop above catches a locale key absent from the source. this catches the
// other direction, which nothing else can see: a key extracted from somewhere it should not
// be, such as a test fixture, is absent from en.json but would be written into all 24
// catalogues by the next translation:generate, and translators would be asked to translate it
const en = require(getLocalePath({ locale: 'en' }));
keys.forEach((key) => {
  if (!Object.keys(en).includes(key)) {
    errors.push(
      `"${key}" was extracted from the source but is missing from the "en" locale dict. ` +
        `If it is a new message, run "translation:generate" and commit the catalogues. ` +
        `If it comes from a test or a fixture, exclude that file from "translation:extract".`
    );
  }
});

// a translation's markup and placeholders have to match the contract of the English string
// it translates. neither the extractor nor the key comparison above looks at a value, so
// without this a broken tag or an invented placeholder ships and the widget silently renders
// that message in English instead of the translation
const PLACEHOLDER = /\{\s*(\w+)\s*\}/g;

function placeholdersIn(message) {
  return new Set(Array.from(message.matchAll(PLACEHOLDER), (match) => match[1]));
}

function tagsIn(message) {
  return new Set(Array.from(message.matchAll(/<(\w+)>/g), (match) => match[1]));
}

// mirrors app/common/intl-message.ts: a self-closing tag is text, and only `</` or `<` opening a
// tag name is markup that has to resolve. testing for any angle bracket instead would
// reject ordinary text such as "under < 10"
const SELF_CLOSING = /<[a-zA-Z][\w-]*\s*\/>/g;
const TAG_START = /<\/|<[a-zA-Z]/;

/**
 * True when anything tag-shaped survives once the pairs the English string uses are taken
 * out. The pair contents are checked too, since stripping a pair removes whatever was
 * nested inside it, and the widget rejects a nested tag.
 */
function hasBrokenMarkup(message, tags) {
  const chunks = [];
  const withoutPairs = tags.reduce(
    (text, tag) =>
      text.replace(new RegExp(`<${tag}>([\\s\\S]*?)</${tag}>`, 'g'), (_match, chunk) => {
        chunks.push(chunk);

        return '';
      }),
    message
  );

  return (
    TAG_START.test(withoutPairs.replace(SELF_CLOSING, '')) ||
    chunks.some((chunk) => TAG_START.test(chunk.replace(SELF_CLOSING, '')))
  );
}

locales.forEach((locale) => {
  if (locale === 'en') {
    return;
  }

  const dict = require(getLocalePath({ locale }));

  Object.keys(dict).forEach((key) => {
    const source = en[key];
    const value = dict[key];

    if (typeof source !== 'string' || typeof value !== 'string') {
      return;
    }

    const tags = Array.from(tagsIn(source));

    if (hasBrokenMarkup(value, tags)) {
      errors.push(
        `"${key}" in "${locale}" has markup the widget cannot resolve: ${JSON.stringify(value)}. ` +
          `Tags must be well-formed pairs of the same names the English string uses ` +
          `(${tags.length ? tags.map((tag) => `<${tag}>`).join(', ') : 'none'}), with no attributes. ` +
          `A translation may leave a tag out entirely. As written, this message renders in English.`
      );
    }

    const unknown = Array.from(placeholdersIn(value)).filter((name) => !placeholdersIn(source).has(name));

    if (unknown.length) {
      errors.push(
        `"${key}" in "${locale}" uses ${unknown.map((name) => `{${name}}`).join(', ')}, which the ` +
          `English string does not provide. As written, this message renders in English.`
      );
    }
  });
});

if (errors.length) {
  console.error(errors.join(`\n`));
  process.exit(1);
}
