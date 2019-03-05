import 'core-js/es7/promise';
import 'focus-visible';

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

    await import(/* webpackChunkName: "polyfills" */ 'core-js').then();
    return;
  };

  const fillFetch = async () => {
    if ('fetch' in window) return;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await import(/* webpackChunkName: "polyfills" */ 'whatwg-fetch' as any).then();
  };

  await Promise.all([fillCoreJs(), fillFetch()]);
  return;
}
