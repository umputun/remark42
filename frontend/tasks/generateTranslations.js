const fs = require('fs');
const path = require('path');

const defaultMessages = require('./defaultMessages');
const getLocalePath = require('./getLocalePath');
const renderLoadLocale = require('./localeLoadTemplate');
const locales = require('./supportedLocales.json');

function removeAbandonedKeys(existKeys, dictionary) {
  return Object.fromEntries(Object.entries(dictionary).filter(([key]) => existKeys.includes(key)));
}

function sortDict(dict) {
  return Object.fromEntries(Object.entries(dict).sort(([a], [b]) => a.localeCompare(b)));
}

locales.forEach(locale => {
  let currentDict = {};
  const pathToDict = getLocalePath(locale);

  if (fs.existsSync(pathToDict)) {
    currentDict = require(pathToDict);
  } else {
    throw Error(`Unable to read translation dictionary for "${locale}" locale`);
  }

  Object.entries(defaultMessages).forEach(([key, defaultMessage]) => {
    if (currentDict[key]) {
      return;
    }

    currentDict[key] = defaultMessage;
  });

  currentDict = removeAbandonedKeys(Object.keys(defaultMessages), currentDict);
  currentDict = sortDict(currentDict);

  fs.writeFileSync(pathToDict, `${JSON.stringify(currentDict, null, 2)}\n`);
  fs.writeFileSync(
    path.resolve(__dirname, '../app/utils/loadLocale.ts'),
    renderLoadLocale(locales.filter(locale => locale !== 'en'))
  );
});
