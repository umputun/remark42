import 'core-js/es6/promise';

// default promise polyfill doesn't include finally
Promise.prototype.finally = function finallyFn(callback) {
  const constructor = this.constructor;

  return this.then(
    (value) => constructor.resolve(callback()).then(() => value),
    (reason) => constructor.resolve(callback()).then(() => reason)
  );
};

window.Promise = Promise;

export default function loadPolyfills() {
  const fillCoreJs = () => {
    if (
      'startsWith' in String.prototype &&
      'endsWith' in String.prototype &&
      'includes' in Array.prototype &&
      'assign' in Object &&
      'keys' in Object
    ) return Promise.resolve();

    return import(/* webpackChunkName: "polyfills" */ 'core-js');
  };

  return fillCoreJs();
}
