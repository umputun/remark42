const spawn = require('child_process').spawn;
const path = require('path');

module.exports = function({ mode = 'es5', glob } = {}) {
  return new Promise((resolve, reject) => {
    if (!glob) {
      reject(new Error('no glob provided'));
    }
    const check = spawn('./node_modules/.bin/es-check', [mode, glob], {
      cwd: path.resolve(__dirname, './'),
    });

    const buffer = [];

    check.stdout.on('data', data => {
      buffer.push(data.toString('utf-8'));
    });

    check.stderr.on('data', data => {
      buffer.push(data.toString('utf-8'));
    });

    check.on('close', code => {
      if (code === 0) resolve();
      reject(new Error(`es-check exited with code ${code}\n\n${buffer.join('\n')}`));
    });
  });
};
