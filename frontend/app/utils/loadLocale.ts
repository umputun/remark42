import enMessages from '../locales/en.json';

export async function loadLocale(locale: string): Promise<Record<string, string>> {
  if (locale === 'ru') {
    return import(/* webpackChunkName: "ru" */ `../locales/ru.json`)
      .then(res => res.default)
      .catch(() => enMessages);
  }
  return enMessages;
}
