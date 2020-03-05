import 'regenerator-runtime/runtime';
import 'es6-promise/auto';
import 'focus-visible';
import '@webcomponents/custom-elements';
import './closest-polyfill';

export default async function loadPolyfills() {
  function fillCoreJs() {
    if (
      'startsWith' in String.prototype &&
      'endsWith' in String.prototype &&
      'includes' in Array.prototype &&
      'assign' in Object &&
      'keys' in Object
    ) {
      return;
    }

    return import(/* webpackChunkName: "core-js" */ 'core-js');
  }

  function fillFetch() {
    if ('fetch' in window) return;

    return import(/* webpackChunkName: "whatwg-fetch" */ 'whatwg-fetch');
  }

  function fillIntersectionObserver() {
    if (
      'IntersectionObserver' in window &&
      'IntersectionObserverEntry' in window &&
      'intersectionRatio' in window.IntersectionObserverEntry.prototype
    ) {
      return;
    }

    return import(/* webpackChunkName: "intersection-observer" */ 'intersection-observer');
  }

  await Promise.all([fillCoreJs(), fillFetch(), fillIntersectionObserver()]);
  return;
}
