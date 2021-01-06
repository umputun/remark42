import { setJsonItem, getJsonItem, updateJsonItem } from './local-storage';

const LS_KEY = 'test';

describe('getJsonItem', () => {
  afterAll(() => {
    localStorage.clear();
  });
  it('should set json to empty localStorage', () => {
    setJsonItem<Record<string, string>>(LS_KEY, {});
    expect(localStorage.getItem(LS_KEY)).toBe('{}');
  });

  it('should update json in localStoeage', () => {
    setJsonItem(LS_KEY, []);
    expect(localStorage.getItem(LS_KEY)).toBe('[]');
  });
});

describe('setJsonItem', () => {
  let consoleSpy: jest.SpyInstance;

  beforeEach(() => {
    consoleSpy = jest.spyOn(console, 'error').mockImplementation();
  });
  afterEach(() => {
    localStorage.clear();
  });

  it('should return null when localStorage is empty', () => {
    expect(getJsonItem(LS_KEY)).toBe(null);
  });

  it('should return value of key', () => {
    localStorage.setItem(LS_KEY, JSON.stringify({}));
    expect(getJsonItem(LS_KEY)).toEqual({});

    localStorage.setItem(LS_KEY, JSON.stringify([]));
    expect(getJsonItem(LS_KEY)).toEqual([]);

    localStorage.setItem(LS_KEY, JSON.stringify(null));
    expect(getJsonItem(LS_KEY)).toBe(null);

    localStorage.setItem(LS_KEY, JSON.stringify(1));
    expect(getJsonItem(LS_KEY)).toBe(1);

    localStorage.setItem(LS_KEY, JSON.stringify(1));
    expect(getJsonItem(LS_KEY)).toBe(1);
  });

  it('should return `null` if value in localStorage is not JSON', () => {
    localStorage.setItem(LS_KEY, '"{:"""');

    expect(getJsonItem(LS_KEY)).toBe(null);
    expect(consoleSpy).toHaveBeenCalled();

    localStorage.setItem(LS_KEY, 'asdas');

    expect(getJsonItem(LS_KEY)).toBe(null);
    expect(consoleSpy).toHaveBeenCalled();
  });
});

describe('updateJsonItem', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('should set data to empty localStorage', () => {
    updateJsonItem(LS_KEY, {});

    expect(localStorage.getItem(LS_KEY)).toBe(JSON.stringify({}));
  });

  it('should update object in localStorage', () => {
    localStorage.setItem(LS_KEY, JSON.stringify({ x: 1 }));
    updateJsonItem(LS_KEY, { y: 1 });

    expect(localStorage.getItem(LS_KEY)).toBe(JSON.stringify({ x: 1, y: 1 }));
  });

  it('should update array in localStorage', () => {
    localStorage.setItem(LS_KEY, JSON.stringify([1, 2, 3]));
    updateJsonItem(LS_KEY, [4, 5, 6]);

    expect(localStorage.getItem(LS_KEY)).toBe(JSON.stringify([1, 2, 3, 4, 5, 6]));
  });

  it('should update data in localStorage with merge', () => {
    localStorage.setItem(LS_KEY, JSON.stringify([3, 4, 5]));
    updateJsonItem(LS_KEY, (data: unknown[]) => [1, 2, ...data]);

    expect(localStorage.getItem(LS_KEY)).toBe(JSON.stringify([1, 2, 3, 4, 5]));
  });
});
