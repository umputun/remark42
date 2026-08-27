import { IS_STORAGE_AVAILABLE } from './constants';
import { readJson, updateJson, writeJson } from './json-store';

/**
 * Where the JSON helpers in json-store.ts meet localStorage, plus the guarded accessors used
 * directly. Storage can be switched off in browser preferences, and every one of these has to keep
 * working when it is: the widget stores nothing it cannot do without.
 */

const failMessage = 'remark42: localStorage access denied, check browser preferences';

export const setItem = IS_STORAGE_AVAILABLE
  ? localStorage.setItem.bind(localStorage)
  : () => {
      console.error(failMessage);
    };

export const getItem = IS_STORAGE_AVAILABLE
  ? localStorage.getItem.bind(localStorage)
  : () => {
      console.error(failMessage);
      return null;
    };

export const removeItem = IS_STORAGE_AVAILABLE
  ? localStorage.removeItem.bind(localStorage)
  : () => {
      console.error(failMessage);
    };

/** The storage the helpers below read and write, honouring the guards above. */
const store = { getItem: (key: string) => getItem(key), setItem: (key: string, value: string) => setItem(key, value) };

export function getJsonItem<T = unknown>(key: string): T | null {
  return readJson<T>(store, key);
}

export function setJsonItem<T = unknown>(key: string, data: T) {
  writeJson(store, key, data);
}

export function updateJsonItem<T = Record<string, unknown>>(key: string, value: (data: T) => T): void;
export function updateJsonItem<T = Record<string, unknown>>(key: string, value: T): void;
export function updateJsonItem<T = unknown[]>(key: string, value: T): void;
export function updateJsonItem<T>(key: string, value: T) {
  updateJson(store, key, value);
}
