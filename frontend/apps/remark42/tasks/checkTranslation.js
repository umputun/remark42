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
        `"${key}" key not found in "${locale}" locale dict. Please run "npm run translation:generate" and commit changes.`
      );
    }
    return null;
  });
});
if (errors.length) {
  // eslint-disable-next-line no-console
  console.error(errors.join(`\n`));
  process.exit(1);
}
