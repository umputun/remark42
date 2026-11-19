let table: Promise<Record<string, string>> | null = null;

/**
 * The name-to-emoji table, fetched on first use and kept for the life of the document.
 *
 * It is a plain file the server embeds and serves from the directory this document came from, so
 * it is asked for relative to the document: the request stays on the document's own origin
 * whatever host the page embedding the widget was configured with.
 */
export function loadEmoji(): Promise<Record<string, string>> {
  if (table === null) {
    table = fetch(new URL('emoji.json', window.location.href).href)
      .then((response) => {
        if (!response.ok) {
          throw new Error(`Remark42: emoji table responded with ${response.status}`);
        }

        return response.json() as Promise<Record<string, string>>;
      })
      .catch((error) => {
        // a load that failed is asked for again the next time a suggestion is wanted
        table = null;
        throw error;
      });
  }

  return table;
}
