import type { Theme, ThemeSetting } from 'common/types';

/** the OS-level color scheme the browser reports through `prefers-color-scheme` */
export function getSystemTheme(): Theme {
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  return 'light';
}

/**
 * Resolves a `theme` config value to the concrete theme to render.
 *
 * `'auto'` follows the OS via `prefers-color-scheme`; a fixed `'light'`/`'dark'` is returned as is.
 */
export function resolveTheme(setting: ThemeSetting): Theme {
  return setting === 'auto' ? getSystemTheme() : setting;
}

/**
 * The `color-scheme` CSS value for a `theme` config value.
 *
 * `'auto'` yields `light dark` so the browser picks a scheme from the OS preference, letting the
 * widget's own colors follow it without a reload; a fixed theme pins the scheme to that value.
 */
export function colorSchemeFor(setting: ThemeSetting): string {
  return setting === 'auto' ? 'light dark' : setting;
}