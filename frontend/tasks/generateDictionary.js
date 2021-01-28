const fs = require('fs');
const path = require('path');
const { keyMessagePairs, keys } = require('./getTranslationKeys');
const { getLocalePath } = require('./getLocalePath');
const { renderLoadLocale } = require('./localeLoadTemplate');
const { getSupportedLocales } = require('./getSupportedLocales');

const locales = getSupportedLocales();

function removeAbandonedKeys(existKeys, dictionary) {
  return Object.fromEntries(Object.entries(dictionary).filter(([key]) => existKeys.includes(key)));
}

function sortDict(dict) {
  return Object.fromEntries(
    Object.entries(dict).sort(([a], [b]) => {
      return a.localeCompare(b);
    })
  );
}

locales.forEach((locale) => {
  let currentDict = {};
  const pathToDict = getLocalePath({ locale });
  if (fs.existsSync(pathToDict)) {
    currentDict = require(pathToDict);
  }
  keyMessagePairs.forEach(([key, defaultMessage]) => {
    if (!currentDict[key] || locale === `en`) {
      currentDict[key] = defaultMessage;
    }
  });
  currentDict = removeAbandonedKeys(keys, currentDict);
  currentDict = sortDict(currentDict);
  fs.writeFileSync(pathToDict, `${JSON.stringify(currentDict, null, 2)}\n`);
  fs.writeFileSync(
    path.resolve(__dirname, `../app/utils/loadLocale.ts`),
    renderLoadLocale(locales.filter((locale) => locale !== 'en'))
  );
});
