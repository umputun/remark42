const htmlDecodeElement = document.createElement('div');

export function htmlUnescape(input: string): string {
  htmlDecodeElement.innerHTML = input;
  return htmlDecodeElement.childNodes.length === 0 ? '' : htmlDecodeElement.childNodes[0].nodeValue || '';
}
