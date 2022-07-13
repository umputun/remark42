import isEqual from 'lodash-es/isEqual';

/**
 * Deeply compares two entities,
 * and if they are equal, then first entity
 * will be returned
 *
 * Useful when we deal with react objects
 */
export function cmpRef<T>(a: T, b: T): T {
  if (isEqual(a, b)) return a;
  return b;
}
