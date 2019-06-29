/* eslint-disable @typescript-eslint/no-explicit-any */

// prettier-ignore
export function throttle(fn: () => any, limit: number): () => void
// prettier-ignore
export function throttle<A>(fn: (a: A) => any, limit: number): (a: A) => void
// prettier-ignore
export function throttle<A, B>(fn: (a: A, b: B) => any, limit: number): (a: A, b: B) => void
// prettier-ignore
export function throttle<A, B, C>(fn: (a: A, b: B, c: C) => any, limit: number): (a: A, b: B, c: C) => void
// prettier-ignore
export function throttle<A, B, C, D>(fn: (a: A, b: B, c: C, d: D) => any, limit: number): (a: A, b: B, c: C, d: D) => void
// prettier-ignore
export function throttle<A, B, C, D, E>(fn: (a: A, b: B, c: C, d: D, e: E) => any, limit: number): (a: A, b: B, c: C, d: D, e: E) => void
export function throttle(func: (...args: any) => any, limit: number = 1000): (...args: any) => void {
  let lastFunc: number;
  let lastRan: number | null = null;
  return function(this: any, ...args: any) {
    const context = this;
    if (!lastRan) {
      func.apply(context, args);
      lastRan = Date.now();
    } else {
      clearTimeout(lastFunc);
      lastFunc = window.setTimeout(() => {
        if (Date.now() - lastRan! >= limit) {
          func.apply(context, args);
          lastRan = Date.now();
        }
      }, limit - (Date.now() - lastRan));
    }
  };
}
