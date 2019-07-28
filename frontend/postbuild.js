/* eslint-disable no-console */

const escheck = require('./escheck');

async function main() {
  try {
    console.log('checking es5 compatibility...');
    escheck({ glob: './public/*.js' });
  } catch (e) {
    console.error(e);
    process.exit(1);
  }
}

main();
