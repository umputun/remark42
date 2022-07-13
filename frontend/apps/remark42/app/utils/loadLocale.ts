/** this is generated file by "npm run translation:generate" **/
// it is ok that is empty. Default messages from code will be used.
const enMessages = {};

export async function loadLocale(locale: string): Promise<Record<string, string>> {
  if (locale === 'ru') {
    return import(/* webpackChunkName: "ru" */ '../locales/ru.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'de') {
    return import(/* webpackChunkName: "de" */ '../locales/de.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'fi') {
    return import(/* webpackChunkName: "fi" */ '../locales/fi.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'es') {
    return import(/* webpackChunkName: "es" */ '../locales/es.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'zh') {
    return import(/* webpackChunkName: "zh" */ '../locales/zh.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'tr') {
    return import(/* webpackChunkName: "tr" */ '../locales/tr.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'bg') {
    return import(/* webpackChunkName: "bg" */ '../locales/bg.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'ua') {
    return import(/* webpackChunkName: "ua" */ '../locales/ua.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'pl') {
    return import(/* webpackChunkName: "pl" */ '../locales/pl.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'vi') {
    return import(/* webpackChunkName: "vi" */ '../locales/vi.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'be') {
    return import(/* webpackChunkName: "be" */ '../locales/be.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'fr') {
    return import(/* webpackChunkName: "fr" */ '../locales/fr.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'ja') {
    return import(/* webpackChunkName: "ja" */ '../locales/ja.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'ko') {
    return import(/* webpackChunkName: "ko" */ '../locales/ko.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'bp') {
    return import(/* webpackChunkName: "bp" */ '../locales/bp.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'it') {
    return import(/* webpackChunkName: "it" */ '../locales/it.json').then((res) => res.default).catch(() => enMessages);
  }
  if (locale === 'ar') {
    return import(/* webpackChunkName: "ar" */ '../locales/ar.json').then((res) => res.default).catch(() => enMessages);
  }

  return enMessages;
}
