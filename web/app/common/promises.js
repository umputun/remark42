import Promise from 'promise-polyfill';

Promise._unhandledRejectionFn = () => {};

Promise.prototype.finally = function finallyFn(callback) {
  const constructor = this.constructor;

  return this.then(
    (value) => constructor.resolve(callback()).then(() => value),
    (reason) => constructor.resolve(callback()).then(() => reason)
  );
};

window.Promise = Promise;
