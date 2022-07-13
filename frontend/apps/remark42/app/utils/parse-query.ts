/** converts window.location.search into object */
export function parseQuery(search: string = window.location.search): Record<string, string> {
  const params: Record<string, string> = {};
  new URLSearchParams(search).forEach((value: string, key: string) => {
    params[key] = value;
  });
  return params;
}
