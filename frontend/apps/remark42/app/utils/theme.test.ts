import { colorSchemeFor, getSystemTheme, resolveTheme } from './theme';

describe('theme utils', () => {
  const originalMatchMedia = window.matchMedia;

  afterEach(() => {
    window.matchMedia = originalMatchMedia;
  });

  function mockMatchMedia(matches: boolean) {
    window.matchMedia = jest.fn().mockImplementation((query: string) => ({
      matches,
      media: query,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
      addListener: jest.fn(),
      removeListener: jest.fn(),
      dispatchEvent: jest.fn(),
      onchange: null,
    })) as unknown as typeof window.matchMedia;
  }

  describe('getSystemTheme', () => {
    it('returns dark when the OS prefers a dark scheme', () => {
      mockMatchMedia(true);
      expect(getSystemTheme()).toBe('dark');
    });

    it('returns light when the OS prefers a light scheme', () => {
      mockMatchMedia(false);
      expect(getSystemTheme()).toBe('light');
    });

    it('returns light when matchMedia is unavailable', () => {
      // @ts-expect-error simulate an environment without matchMedia
      delete window.matchMedia;
      expect(getSystemTheme()).toBe('light');
    });
  });

  describe('resolveTheme', () => {
    it('passes a fixed theme through unchanged', () => {
      expect(resolveTheme('dark')).toBe('dark');
      expect(resolveTheme('light')).toBe('light');
    });

    it('resolves auto to the OS theme', () => {
      mockMatchMedia(true);
      expect(resolveTheme('auto')).toBe('dark');
      mockMatchMedia(false);
      expect(resolveTheme('auto')).toBe('light');
    });
  });

  describe('colorSchemeFor', () => {
    it('pins a fixed theme to itself', () => {
      expect(colorSchemeFor('dark')).toBe('dark');
      expect(colorSchemeFor('light')).toBe('light');
    });

    it('lets the browser pick for auto', () => {
      expect(colorSchemeFor('auto')).toBe('light dark');
    });
  });
});