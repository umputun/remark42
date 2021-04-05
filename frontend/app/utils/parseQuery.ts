/** converts widnow.location.search into object */

export default function parseQuery<T extends {}>(search: string = window.location.search): T {
  const params: { [key: string]: string } = {};
  new URLSearchParams(search).forEach((value: string, key: string) => {
    params[key] = value;
  });
  return params as T;
}
