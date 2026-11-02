import { act, renderHook } from '@testing-library/preact';

import { useSessionStorage } from './useSessionState';

describe('useSessionStorage', () => {
  // the tests write under the same key, so without this they depend on each other's order
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('should return a value and a setter', () => {
    const { result } = renderHook(() => useSessionStorage('test', 0));
    expect(result.current).toHaveLength(2);
    expect(result.current[0]).toBe(0);
    expect(result.current[1]).toBeInstanceOf(Function);
  });

  it('should store the value it is given', () => {
    const { result } = renderHook(() => useSessionStorage('test', 0));

    act(() => result.current[1](5));

    expect(result.current[0]).toBe(5);
    expect(sessionStorage.getItem('test')).toBe('5');
  });
});
