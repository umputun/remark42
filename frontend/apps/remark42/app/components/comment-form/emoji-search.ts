export type Emoji = {
  key: string;
  emoji: string;
};

/**
 * Drops the colons a reader may have typed around a name: a trailing colon goes, and so does
 * everything up to and including a colon anywhere else.
 */
function stripColons(text: string): string {
  const index = text.indexOf(':');

  if (index === -1) {
    return text;
  }

  return stripColons(index === text.length - 1 ? text.substring(0, index) : text.substring(index + 1));
}

/**
 * Emoji whose name starts with the text, in the order the table lists them.
 *
 * The table maps a name to its emoji and is the file the server embeds as `emoji.json`.
 */
export function searchEmoji(table: Record<string, string>, text: string): Emoji[] {
  const prefix = stripColons(text);

  return Object.keys(table)
    .filter((key) => key.startsWith(prefix))
    .map((key) => ({ key, emoji: table[key] }));
}
