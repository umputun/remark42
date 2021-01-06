const path = require('path');

module.exports = function getLocalePath(locale) {
  if (locale === 'en') {
    return path.resolve(__dirname, '../extracted-messages/messages.json');
  }

  return path.resolve(__dirname, `../app/locales/${locale}.json`);
};
