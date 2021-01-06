/** this is generated file by "npm run translation:generate" **/
// it is ok that is empty. Default messages from code will be used.
const enMessages = {};

export async function loadLocale(locale: string): Promise<Record<string, string>> {
  if (locale === 'ru') {
    return import(/* webpackChunkName: "ru.locale" */ '../locales/ru.json')
      .then(m => m.default)
      .catch(() => enMessages);
  }
  if (locale === 'de') {
    return import(/* webpackChunkName: "de.locale" */ '../locales/de.json')
      .then(m => m.default)
      .catch(() => enMessages);
  }
  if (locale === 'fi') {
    return import(/* webpackChunkName: "fi.locale" */ '../locales/fi.json')
      .then(m => m.default)
      .catch(() => enMessages);
  }
  if (locale === 'es') {
    return import(/* webpackChunkName: "es.locale" */ '../locales/es.json')
      .then(m => m.default)
      .catch(() => enMessages);
  }
  if (locale === 'zh') {
    return import(/* webpackChunkName: "zh.locale" */ '../locales/zh.json')
      .then(m => m.default)
      .catch(() => enMessages);
  }
  if (locale === 'tr') {
    return import(/* webpackChunkName: "tr.locale" */ '../locales/tr.json')
      .then(m => m.default)
      .catch(() => enMessages);
  }
  if (locale === 'bg') {
    return import(/* webpackChunkName: "bg.locale" */ '../locales/bg.json')
      .then(m => m.default)
      .catch(() => enMessages);
  }
  if (locale === 'ua') {
    return import(/* webpackChunkName: "ua.locale" */ '../locales/ua.json')
      .then(m => m.default)
      .catch(() => enMessages);
  }
  if (locale === 'pl') {
    return import(/* webpackChunkName: "pl.locale" */ '../locales/pl.json')
      .then(m => m.default)
      .catch(() => enMessages);
  }

  return enMessages;
}
