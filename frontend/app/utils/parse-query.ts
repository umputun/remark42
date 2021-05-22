/** converts window.location.search into object */

export function parseQuery<T extends Record<string, string>>(search: string = window.location.search): T {
  const params: { [key: string]: string } = {};
  new URLSearchParams(search).forEach((value: string, key: string) => {
    params[key] = value;
  });
  return params as T;
}
