import { renderHook } from '@testing-library/preact-hooks';

import { useSessionStorage } from './useSessionState';

describe('useSessionStorage', () => {
  it('should return a value and a setter', () => {
    const { result } = renderHook(() => useSessionStorage('test', 0));
    expect(result.current!).toHaveLength(2);
    expect(result.current![0]).toBe(0);
    expect(result.current![1]).toBeInstanceOf(Function);
  });

  it('should return the initial value', () => {
    const { result } = renderHook(() => useSessionStorage('test', 0));
    expect(result.current![0]).toBe(0);
  });

  it('should return the stored value', () => {
    sessionStorage.setItem('test', JSON.stringify(1));
    const { result } = renderHook(() => useSessionStorage('test', 0));
    expect(result.current![0]).toBe(1);
  });

  it('should return the stored value if it is falsy', () => {
    sessionStorage.setItem('test', JSON.stringify(false));
    const { result } = renderHook(() => useSessionStorage('test', 0));
    expect(result.current![0]).toBe(false);
  });

  it('should return the initial value if the stored value is not valid JSON', () => {
    sessionStorage.setItem('test', 'not valid JSON');
    const { result } = renderHook(() => useSessionStorage('test', 0));
    expect(result.current![0]).toBe(0);
  });

  it('should return null if the stored value is null', () => {
    // @ts-ignore
    sessionStorage.setItem('test', null);
    const { result } = renderHook(() => useSessionStorage('test', 0));
    expect(result.current![0]).toBe(null);
  });

  it('should return the initial value if the stored value is undefined', () => {
    // @ts-ignore
    sessionStorage.setItem('test', undefined);
    const { result } = renderHook(() => useSessionStorage('test', 0));
    expect(result.current![0]).toBe(0);
  });
});
