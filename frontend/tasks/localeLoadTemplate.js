function renderLoadLocale(locales) {
  return `/** this is generated file by "npm run translation:generate" **/
// it is ok that is empty. Default messages from code will be used.
const enMessages = {};

export async function loadLocale(locale: string): Promise<Record<string, string>> {
${locales
  .map(
    (locale) => `  if (locale === '${locale}') {
    return import(/* webpackChunkName: "${locale}" */ '../locales/${locale}.json').then((res) => res.default).catch(() => enMessages);
  }
`
  )
  .join('')}
  return enMessages;
}\n`;
}

module.exports = { renderLoadLocale };
