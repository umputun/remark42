import supportedLocales from '../../tasks/supportedLocales.json';
import { BASE_URL } from 'common/constants.config';

// the catalogues are plain files among the assets embedded in the binary, served as
// /web/<locale>.json. English is never fetched: the messages in the code are the English text,
// and en.json is only the copy of them that translators work from
const catalogues = new Set(supportedLocales.filter((locale) => locale !== 'en'));

/**
 * Fetches the catalogue for a locale, or resolves to an empty one, which makes every message fall
 * back to English. The locale comes from the host page, so only a name on the list of supported
 * catalogues ever becomes part of a URL. last-comments runs on the host page, which makes this a
 * cross-origin request there: the server's CORS headers cover it, or with --proxy-cors the proxy's.
 */
export async function loadLocale(locale: string): Promise<Record<string, string>> {
  if (!catalogues.has(locale)) {
    return {};
  }
  try {
    const response = await fetch(`${BASE_URL}/web/${locale}.json`);
    if (!response.ok) {
      return {};
    }
    return await response.json();
  } catch {
    return {};
  }
}
