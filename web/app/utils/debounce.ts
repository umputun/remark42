/* eslint-disable @typescript-eslint/no-explicit-any */

// type signature is wrong, but there is no way
// in typescript to augment function return type currently
export default function debounce<F extends (...args: any[]) => void>(func: F, wait: number): F {
  let timeout: number | undefined;
  return function(this: any): void {
    const laterCall = (): void => func.apply(this, arguments as any);
    window.clearTimeout(timeout);
    timeout = window.setTimeout(laterCall, wait);
  } as any;
}
