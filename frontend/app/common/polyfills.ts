import 'regenerator-runtime/runtime';
import 'es6-promise/auto';
import 'focus-visible';
import '@webcomponents/custom-elements';
import './closest-polyfill';

export default async function loadPolyfills() {
  const fillCoreJs = async () => {
    if (
      'startsWith' in String.prototype &&
      'endsWith' in String.prototype &&
      'includes' in Array.prototype &&
      'assign' in Object &&
      'keys' in Object
    )
      return;

    await import(/* webpackChunkName: "core-js" */ 'core-js').then();
    return;
  };

  const fillFetch = async () => {
    if ('fetch' in window) return;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await import(/* webpackChunkName: "whatwg-fetch" */ 'whatwg-fetch' as any).then();
  };

  const fillIntersectionObserver = async () => {
    if (
      'IntersectionObserver' in window &&
      'IntersectionObserverEntry' in window &&
      'intersectionRatio' in window.IntersectionObserverEntry.prototype
    ) {
      return;
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await import(/* webpackChunkName: "intersection-observer" */ 'intersection-observer' as any).then();
  };

  await Promise.all([fillCoreJs(), fillFetch(), fillIntersectionObserver()]);
  return;
}
