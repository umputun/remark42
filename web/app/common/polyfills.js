import 'core-js/es7/promise';

export default function loadPolyfills() {
  const fillCoreJs = () => {
    if (
      'startsWith' in String.prototype &&
      'endsWith' in String.prototype &&
      'includes' in Array.prototype &&
      'assign' in Object &&
      'keys' in Object
    )
      return Promise.resolve();

    return import(/* webpackChunkName: "polyfills" */ 'core-js');
  };

  return fillCoreJs();
}
