/** converts widnow.location.search into object */
export function parseQuery(search: string): { [key: string]: string } {
  if (search.length < 2) return {};
  return search
    .substr(1)
    .split('&')
    .map(
      (chunk): [string, string] => {
        const parts = chunk.split('=');
        if (parts.length < 2) {
          parts[1] = '';
        } else {
          parts[1] = decodeURIComponent(parts[1]);
        }
        return parts as [string, string];
      }
    )
    .reduce<{ [key: string]: string }>((c, x) => {
      c[x[0]] = x[1];
      return c;
    }, {});
}
