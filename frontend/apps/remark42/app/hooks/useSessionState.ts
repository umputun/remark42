import type { StateUpdater, Dispatch } from 'preact/hooks';
import { useState } from 'preact/hooks';

import { readStored, resolveUpdate, writeStored } from './session-storage';

/**
 * State backed by sessionStorage. The reading, writing and update resolution live in
 * session-storage.ts, which knows nothing about preact and carries the cases worth testing.
 */
function useSessionStorage<T>(key: string, initialValue?: T): [T, Dispatch<StateUpdater<T>>] {
  const [storedValue, setStoredValue] = useState<T>(() => readStored(sessionStorage, key, initialValue) as T);

  const setValue: typeof setStoredValue = (value) => {
    const valueToStore = resolveUpdate(value as T | ((current: T) => T), storedValue);

    setStoredValue(valueToStore);
    writeStored(sessionStorage, key, valueToStore);
  };

  return [storedValue, setValue];
}

export { useSessionStorage };
