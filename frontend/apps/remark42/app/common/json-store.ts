import { isObject } from '../utils/is-object.ts';

/**
 * Reading, writing and merging JSON in a web storage, with the storage passed in.
 *
 * Kept apart from local-storage.ts so it can be tested without a browser: that module binds
 * localStorage at import time, through a constant that probes it, so importing the file at all
 * needs one. The cases worth having are the ones a browser test cannot arrange on purpose, since
 * they are about what anything else with access to the key may have left behind.
 */

/** The part of the Storage interface used here, so a plain object satisfies it in a test. */
export type JsonStore = Pick<Storage, 'getItem' | 'setItem'>;

/**
 * Reads a stored value, or null when there is nothing usable there.
 *
 * A missing key and unparsable content give the same answer, since neither leaves the caller
 * anything to work with, but only the second is worth reporting: it means something wrote to this
 * key that this widget cannot read.
 */
export function readJson<T = unknown>(storage: JsonStore, key: string): T | null {
  try {
    const json = storage.getItem(key);

    if (json === null) {
      return null;
    }

    return JSON.parse(json);
  } catch (e) {
    console.error(`remark42: error on read JSON from ${key} in localStorage`, e);
    return null;
  }
}

/**
 * Writes a value as JSON.
 *
 * Failure is reported and swallowed instead of thrown: everything kept here is a preference, and
 * losing one is not worth taking down the caller that was storing it.
 */
export function writeJson<T = unknown>(storage: JsonStore, key: string, data: T): void {
  try {
    storage.setItem(key, JSON.stringify(data));
  } catch (e) {
    console.error(`remark42: error on parse JSON from ${key} in localStorage`, e);
  }
}

/**
 * Merges into a stored value: an array is appended to, an object is spread over, and a function is
 * applied to whatever is there.
 *
 * Which of the three happens is decided by the argument and not by what is stored, so appending an
 * array to a key holding an object throws instead of replacing it. That is deliberate as far as
 * the caller is concerned, since the two shapes are never the same key.
 */
export function updateJson<T>(storage: JsonStore, key: string, value: T | ((data: T) => T)): void {
  const savedData = readJson<T>(storage, key);

  if (Array.isArray(value) && Array.isArray(savedData)) {
    writeJson(storage, key, [...savedData, ...value]);
    return;
  }

  if (isObject(value)) {
    writeJson(storage, key, { ...savedData, ...value });
    return;
  }

  if (typeof value === 'function') {
    writeJson(storage, key, (value as (data: T | null) => T)(savedData));
    return;
  }

  throw new Error(`remark42: error on update JSON for ${key} in localStorage`);
}
