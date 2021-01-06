const fs = require('fs');
const process = require('process');
const locales = require('./supportedLocales');
const getLocalePath = require('./getLocalePath');
const defaultMessages = require('./defaultMessages');

const errors = [];

locales.forEach(locale => {
  const pathToDict = getLocalePath(locale);

  if (!fs.existsSync(pathToDict)) {
    throw new Error(
      `"${locale}" translation file not found. Please run "npm run translation:generate" and commit changes.`
    );
  }

  const dict = require(pathToDict);
  const keysFromDict = Object.keys(dict);
  const keys = Object.keys(defaultMessages);

  keysFromDict.forEach(key => {
    if (!keys.includes(key)) {
      errors.push(
        `"${key}" key not found in "${locale}" locale dict. Please run "npm run translation:generate" and commit changes.`
      );
    }
  });
});

if (errors.length) {
  // eslint-disable-next-line no-console
  console.error(errors.join(`\n`));
  process.exit(1);
}
