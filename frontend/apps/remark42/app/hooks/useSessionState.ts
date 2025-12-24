import { useState, StateUpdater } from 'preact/hooks';

function useSessionStorage<T>(key: string, initialValue?: T): [T, StateUpdater<T>] {
  const [storedValue, setStoredValue] = useState<T>(() => {
    const item = sessionStorage.getItem(key);
    if (item === null) {
      return initialValue;
    }
    try {
      return JSON.parse(item);
    } catch {
      return initialValue;
    }
  });
  const setValue: typeof setStoredValue = (value) => {
    const valueToStore = value instanceof Function ? value(storedValue) : value;
    setStoredValue(valueToStore);
    sessionStorage.setItem(key, JSON.stringify(valueToStore));
  };
  return [storedValue, setValue];
}

export { useSessionStorage };
