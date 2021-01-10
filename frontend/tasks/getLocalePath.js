const path = require('path');
module.exports = {
  getLocalePath({ locale }) {
    return path.resolve(__dirname, `../app/locales/${locale}.json`);
  },
};
