export async function loadLocale(locale: string): Promise<Record<string, string>> {
  if (locale === 'ru') {
    return import(/* webpackChunkName: "ru" */ `../locales/ru.json`).then(res => res.default);
  }
  return {};
}
