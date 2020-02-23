function renderLoadLocale(locales) {
  return `/** this is generated file by "npm run generate-langs" **/
import enMessages from '../locales/en.json';

export async function loadLocale(locale: string): Promise<Record<string, string>> {
${locales
  .map(
    locale => `  if (locale === '${locale}') {
    return import(/* webpackChunkName: "${locale}" */ '../locales/${locale}.json')
      .then(res => res.default)
      .catch(() => enMessages);
  }
`
  )
  .join('')}
  return enMessages;
}\n`;
}

module.exports = { renderLoadLocale };
