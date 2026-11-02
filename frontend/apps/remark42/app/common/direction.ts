/**
 * Maps a locale to its writing direction.
 *
 * The set below is every RTL locale this widget ships a catalogue for today (`ar`, `fa`, `he`);
 * add a code here when a new RTL catalogue is added under `app/locales/`. Anything else, known or
 * not, defaults to `ltr` — there is no reliable way to derive direction from an arbitrary BCP 47
 * tag without a much larger locale database than this widget otherwise needs.
 */
const RTL_LOCALES: ReadonlySet<string> = new Set(['ar', 'fa', 'he']);

export type Direction = 'ltr' | 'rtl';

export function getDirection(locale: string): Direction {
  return RTL_LOCALES.has(locale) ? 'rtl' : 'ltr';
}
