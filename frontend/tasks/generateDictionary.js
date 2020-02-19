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
  fs.writeFileSync(pathToDict, JSON.stringify(removeAbandonedKeys(keysSet, currentDict), null, 2) + '\n');
});
