const fs = require('fs');
const path = require('path');
const defaultMessages = require('../extracted-messages/messages');

const locales = ['en', 'ru'];

const keyMessagePairs = [];
const keysSet = new Set();
defaultMessages.forEach(({ id, defaultMessage }) => {
  keyMessagePairs.push([id, defaultMessage]);
  keysSet.add(id);
});

function removeAbandonedKeys(existKeys, dictionary) {
  return Object.fromEntries(Object.entries(dictionary).filter(([key]) => existKeys.has(key)));
}

function sortDict(dict) {
  return Object.fromEntries(
    Object.entries(dict).sort(([a], [b]) => {
      return a.localeCompare(b);
    })
  );
}

locales.forEach(locale => {
  let currentDict = {};
  const pathToDict = path.resolve(__dirname, `../app/locales/${locale}.json`);
  if (fs.existsSync(pathToDict)) {
    currentDict = require(pathToDict);
  }
  keyMessagePairs.forEach(([key, defaultMessage]) => {
    if (!currentDict[key] || locale === `en`) {
      currentDict[key] = defaultMessage;
    }
  });
  currentDict = removeAbandonedKeys(keysSet, currentDict);
  currentDict = sortDict(currentDict);
  fs.writeFileSync(pathToDict, JSON.stringify(currentDict, null, 2) + '\n');
});
