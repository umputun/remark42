/* eslint-disable @typescript-eslint/no-explicit-any */

export default function debounce<T extends any[]>(
  fn: (...args: T) => unknown,
  wait: number = 1000
): (...args: Parameters<typeof fn>) => void {
  let timeout: number | undefined;
  return function (this: any, ...args): void {
    const laterCall = (): unknown => fn.apply(this, args);
    window.clearTimeout(timeout);
    timeout = window.setTimeout(laterCall, wait);
  };
}
