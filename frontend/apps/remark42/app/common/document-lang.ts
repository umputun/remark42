/** catalogue names that are not the BCP 47 tag of their language */
const LANGUAGE_TAGS: Readonly<Record<string, string>> = {
  ua: 'uk',
  bp: 'pt-BR',
  'zh-tw': 'zh-TW',
};

/**
 * The `lang` value of a locale whose catalogue the widget shows. A locale without one shows English
 * messages, and the caller reads it as English without asking here.
 */
export function getDocumentLang(locale: string): string {
  return LANGUAGE_TAGS[locale] ?? locale;
}
