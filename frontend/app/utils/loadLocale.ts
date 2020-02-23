/** this is generated file by "npm run generate-langs" **/
import enMessages from '../locales/en.json';

export async function loadLocale(locale: string): Promise<Record<string, string>> {
  if (locale === 'ru') {
    return import(/* webpackChunkName: "ru" */ '../locales/ru.json')
      .then(res => res.default)
      .catch(() => enMessages);
  }
  if (locale === 'de') {
    return import(/* webpackChunkName: "de" */ '../locales/de.json')
      .then(res => res.default)
      .catch(() => enMessages);
  }

  return enMessages;
}
