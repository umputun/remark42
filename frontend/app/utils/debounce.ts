type FnType<T extends unknown[]> = (...args: T) => unknown;

export default function debounce<T extends unknown[]>(
  fn: FnType<T>,
  wait = 1000
): (...args: Parameters<FnType<T>>) => void {
  let timeout: number | undefined;

  return function (this: unknown, ...args): void {
    const laterCall = (): unknown => fn.apply(this, args);
    window.clearTimeout(timeout);
    timeout = window.setTimeout(laterCall, wait);
  };
}
