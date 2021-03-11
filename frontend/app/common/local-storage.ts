import { IS_STORAGE_AVAILABLE } from './constants';

const failMessage = 'remark42: localStorage access denied, check browser preferences';

export const setItem = IS_STORAGE_AVAILABLE
  ? localStorage.setItem.bind(localStorage)
  : () => {
      console.error(failMessage); // eslint-disable-line no-console
    };

export const getItem = IS_STORAGE_AVAILABLE
  ? localStorage.getItem.bind(localStorage)
  : () => {
      console.error(failMessage); // eslint-disable-line no-console
      return null;
    };

export const removeItem = IS_STORAGE_AVAILABLE
  ? localStorage.removeItem.bind(localStorage)
  : () => {
      console.error(failMessage); // eslint-disable-line no-console
    };

export function getJsonItem<T = unknown>(key: string): T | null {
  try {
    const json = getItem(key);

    if (json === null) {
      return null;
    }

    const data = JSON.parse(json);

    return data;
  } catch (e) {
    console.error(`remark42: error on read JSON from ${key} in localStorage`, e); // eslint-disable-line no-console
    return null;
  }
}

export function setJsonItem<T = unknown>(key: string, data: T) {
  try {
    setItem(key, JSON.stringify(data));
  } catch (e) {
    console.error(`remark42: error on parse JSON from ${key} in localStorage`, e); // eslint-disable-line no-console
  }
}

export function updateJsonItem<T = Record<string, unknown>>(key: string, value: (data: T) => T): void;
export function updateJsonItem<T = Record<string, unknown>>(key: string, value: T): void;
export function updateJsonItem<T = unknown[]>(key: string, value: T): void;
export function updateJsonItem<T>(key: string, value: T) {
  const savedData = getJsonItem<T>(key);

  if (Array.isArray(value) && Array.isArray(savedData)) {
    setJsonItem(key, [...savedData, ...value]);
    return;
  }

  if (value !== null && typeof value === 'object') {
    setJsonItem(key, { ...savedData, ...value });
    return;
  }

  if (typeof value === 'function') {
    setJsonItem(key, value(savedData));
    return;
  }

  throw new Error(`remark42: error on update JSON for ${key} in localStorage`);
}
