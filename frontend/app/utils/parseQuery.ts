/** converts widnow.location.search into object */
export default function URLSearchParams<T extends {}>(search: string = window.location.search): T {
  if (search.length < 2) {
    return {} as T;
  }

  return search
    .substr(1)
    .split('&')
    .reduce((accum, param) => {
      const [key, value] = param.split('=');

      return {
        ...accum,
        [key]: value ? decodeURIComponent(value) : '',
      };
    }, {} as T);
}
