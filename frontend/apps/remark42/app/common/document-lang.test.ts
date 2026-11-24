import { getDocumentLang } from './document-lang';

describe('getDocumentLang', () => {
  it.each(['en', 'ru', 'de', 'ar', 'he'])('keeps %s', (locale) => {
    expect(getDocumentLang(locale)).toBe(locale);
  });

  it.each([
    ['ua', 'uk'],
    ['bp', 'pt-BR'],
    ['zh-tw', 'zh-TW'],
  ])('writes the catalogue %s as the language tag %s', (locale, lang) => {
    expect(getDocumentLang(locale)).toBe(lang);
  });
});
