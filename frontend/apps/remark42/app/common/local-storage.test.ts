import { getJsonItem, setJsonItem } from './local-storage';

/**
 * The decoding and merging live in `json-store.ts` and are covered there against a storage handed
 * in. What only a browser can show is that these wrappers reach the real `localStorage`.
 */
describe('the localStorage wrappers', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('round-trips a value through the browser store', () => {
    setJsonItem('wiring', { a: 1 });

    expect(localStorage.getItem('wiring')).toBe('{"a":1}');
    expect(getJsonItem('wiring')).toEqual({ a: 1 });
  });
});
