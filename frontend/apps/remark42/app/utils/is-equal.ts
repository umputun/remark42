/**
 * Deep equality over the values the API returns as JSON: primitives, arrays and plain objects.
 *
 * Anything else, a Date or a Map or a class instance, is equal only to itself. Comparing such a
 * value by its own keys would call two different dates equal, since neither has any, and the
 * caller would keep a stale value; reporting a difference only costs a render.
 */
export function isEqual(a: unknown, b: unknown): boolean {
  // NaN is the one value that does not equal itself
  if (a === b || (Number.isNaN(a) && Number.isNaN(b))) {
    return true;
  }
  if (typeof a !== 'object' || typeof b !== 'object' || a === null || b === null) {
    return false;
  }

  const proto = Object.getPrototypeOf(a);

  if (proto !== Object.getPrototypeOf(b) || (proto !== Object.prototype && proto !== Array.prototype)) {
    return false;
  }

  const keys = Object.keys(a);

  return (
    keys.length === Object.keys(b).length &&
    keys.every(
      (key) =>
        Object.prototype.hasOwnProperty.call(b, key) &&
        isEqual((a as Record<string, unknown>)[key], (b as Record<string, unknown>)[key])
    )
  );
}
