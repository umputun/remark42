import Promise from 'promise-polyfill';

/* eslint-disable no-underscore-dangle */
Promise._unhandledRejectionFn = () => {};
/* eslint-enable no-underscore-dangle */

/* eslint-disable no-extend-native */
Promise.prototype.finally = function finallyFn(callback) {
  const constructor = this.constructor;

  return this.then(
    (value) => constructor.resolve(callback()).then(() => value),
    (reason) => constructor.resolve(callback()).then(() => reason)
  );
};
/* eslint-enable no-extend-native */

window.Promise = Promise;
