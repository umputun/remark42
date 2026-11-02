/**
 * The reading and writing behind useSessionStorage, with no preact in it.
 *
 * Kept apart from the hook so the part that can be wrong is testable as what it is: a decoder over
 * whatever sessionStorage happens to hold. The interesting cases are the ones a browser test cannot
 * produce on purpose -- a value that is not JSON, a stored `null`, a key that was never written --
 * and each of them decides whether a caller gets its initial value or something it cannot use.
 *
 * `storage` is passed in instead of being read from the global, so a test needs no browser and the hook
 * stays the only place that knows about `sessionStorage`.
 */

/** The part of the Storage interface used here, so a plain object satisfies it in a test. */
export type SessionStore = Pick<Storage, 'getItem' | 'setItem'>;

/**
 * Reads a stored value, falling back to initial whenever the stored one cannot be used.
 *
 * A missing key and unparsable content are the same answer on purpose: both mean there is nothing
 * to restore. A stored `null` is not the same, and is returned, since a caller that wrote null
 * meant it.
 */
export function readStored<T>(storage: SessionStore, key: string, initial?: T): T | undefined {
  const item = storage.getItem(key);

  if (item === null) {
    return initial;
  }

  try {
    return JSON.parse(item) as T;
  } catch {
    return initial;
  }
}

/** Writes a value under key, in the form readStored expects to find. */
export function writeStored<T>(storage: SessionStore, key: string, value: T): void {
  storage.setItem(key, JSON.stringify(value));
}

/** Resolves a state update, which callers may pass either as a value or as a function of the current one. */
export function resolveUpdate<T>(update: T | ((current: T) => T), current: T): T {
  return update instanceof Function ? update(current) : update;
}
