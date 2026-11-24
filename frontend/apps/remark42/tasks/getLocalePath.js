const path = require('path');
module.exports = {
  getLocalePath({ locale }) {
    // the catalogues are served as plain files from the assets embedded in the backend binary
    return path.resolve(__dirname, `../../../../backend/app/webassets/assets/${locale}.json`);
  },
};
