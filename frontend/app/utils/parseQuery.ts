/** converts widnow.location.search into object */
export default function parseQuery(search: string = window.location.search) {
  if (search.length < 2) return {};

  return search
    .substr(1)
    .split('&')
    .reduce((accum, param) => {
      const [key, value] = param.split('=');

      return {
        ...accum,
        [key]: value ? decodeURIComponent(value) : '',
      };
    }, {} as Record<string, string>);
}
