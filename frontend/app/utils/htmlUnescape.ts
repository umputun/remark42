const htmlDecodeElement = document.createElement('div');

export function htmlUnescape(input: string): string {
  htmlDecodeElement.innerHTML = input;
  return htmlDecodeElement.childNodes.length === 0 ? '' : htmlDecodeElement.childNodes[0].nodeValue || '';
}

const replaceRegexes: ([RegExp, string])[] = [[/&#34;/g, '"'], [/&#39;/g, "'"], [/&amp;/g, '&']];

/**
 * Performs partial unescape for [", ', &] symbols
 */
export function htmlPartialUnescape(input: string): string {
  let out = input;
  for (const op of replaceRegexes) {
    out = out.replace(op[0], op[1]);
  }
  return out;
}
