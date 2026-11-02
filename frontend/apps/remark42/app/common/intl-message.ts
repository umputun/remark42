/**
 * Message formatting for the widget's i18n binding, with no preact in it.
 *
 * Kept apart from intl.tsx so it can be tested as what it is: a parser over strings. The binding
 * around it needs a renderer and a component tree; this needs neither, and the cases worth having
 * are the ones no shipped catalogue can produce -- an unclosed tag, a tag carrying attributes, a
 * brace that is not a placeholder -- which is exactly the matrix a browser test cannot reach.
 *
 * `ComponentChildren` is imported as a type alone. It erases at build time, so this module pulls in
 * nothing at runtime and a test runner needs no packages installed to load it.
 *
 * What this deliberately does not implement, none of which any of the 24 catalogues uses: ICU
 * plural, select and selectordinal; typed arguments such as `{n, number}` or `{d, date}`; and ICU
 * apostrophe quoting. A plural or select form carries braces the placeholder rule rejects, so it
 * falls back wherever values are passed and renders raw where they are not. Quoting is not
 * interpreted at all: `''` stays two apostrophes and `'{name}'` interpolates with the quotes in
 * place. Add support before accepting one.
 *
 * A message the binding cannot parse falls back to the message in the source: a broken, unhandled
 * or nested tag, a brace that is not a well-formed placeholder, and a placeholder naming a value
 * the caller did not supply.
 */
import type { ComponentChildren } from 'preact';

export type MessageDescriptor = {
  id: string;
  defaultMessage?: string;
  description?: string;
};

/** A rich-text handler, wrapping the text between a matching pair of tags. */
type ChunkFormatter = (chunk: string) => ComponentChildren;

type PrimitiveValues = Record<string, string | number>;
export type MessageValues = Record<string, string | number | ChunkFormatter>;

export type IntlShape = {
  locale: string;
  formatMessage(descriptor: MessageDescriptor, values?: PrimitiveValues): string;
  formatMessage(descriptor: MessageDescriptor, values: MessageValues): ComponentChildren;
  formatDate(value: Date | number, options?: Intl.DateTimeFormatOptions): string;
  formatTime(value: Date | number, options?: Intl.DateTimeFormatOptions): string;
};

/**
 * Identity function. Its first type parameter is the key type instead of the record,
 * so callers can constrain keys explicitly, as in `defineMessages<string>`.
 */
export function defineMessages<K extends PropertyKey, T = MessageDescriptor, U extends Record<K, T> = Record<K, T>>(
  messages: U
): U {
  return messages;
}

const PLACEHOLDER = /\{\s*(\w+)\s*\}/g;

/** Replaces `{name}`, tolerating whitespace inside the braces, and leaves a placeholder without a value untouched. */
function interpolate(message: string, values: MessageValues): string {
  return message.replace(PLACEHOLDER, (match, key: string) => {
    const value = values[key];

    return value === undefined || typeof value === 'function' ? match : String(value);
  });
}

/** Escapes a tag name so a metacharacter in it cannot change or break the pattern. */
function escapeForRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** Matches a well-formed pair for any of the given tag names, capturing the name and the chunk. */
function pairMatcher(tags: string[], flags = ''): RegExp {
  return new RegExp(`<(${tags.map(escapeForRegExp).join('|')})>([\\s\\S]*?)</\\1>`, flags);
}

/** A self-closing tag is text, not markup, matching formatjs. */
const SELF_CLOSING = /<[a-zA-Z][\w-]*\s*\/>/g;

/** Any `</`, or a `<` opening a tag name, is markup that has to resolve. */
const TAG_START = /<\/|<[a-zA-Z]/;

/**
 * True when a brace is not part of a well-formed placeholder, or a well-formed one names
 * a value the caller did not supply. Both are errors, and the message falls back.
 */
function hasBrokenPlaceholders(message: string, values: MessageValues): boolean {
  if (/[{}]/.test(message.replace(PLACEHOLDER, ''))) {
    return true;
  }

  for (const match of message.matchAll(PLACEHOLDER)) {
    if (!(match[1] in values)) {
      return true;
    }
  }

  return false;
}

/**
 * True when anything tag-shaped survives once the handled pairs are taken out.
 *
 * A broken tag, an unhandled one, or one nested inside another makes the whole string
 * fall back to the source message instead of reaching the page as raw markup. The
 * chunks are checked as well as the text around them, because stripping a pair
 * removes whatever was nested inside it.
 */
function hasBrokenMarkup(message: string, tags: string[]): boolean {
  const chunks: string[] = [];
  // an empty tag list would build an empty alternation, which matches `<></>`
  const withoutPairs =
    tags.length === 0
      ? message
      : message.replace(pairMatcher(tags, 'g'), (_match, _name: string, chunk: string) => {
          chunks.push(chunk);

          return '';
        });
  const residue = withoutPairs.replace(SELF_CLOSING, '');

  return TAG_START.test(residue) || chunks.some((chunk) => TAG_START.test(chunk.replace(SELF_CLOSING, '')));
}

/** True when the message cannot be parsed and the source message should be used instead. */
function isBroken(message: string, values: MessageValues, tags: string[]): boolean {
  return hasBrokenMarkup(message, tags) || hasBrokenPlaceholders(message, values);
}

/** Splits a message into text and the results of its rich-text handlers. */
function formatRich(message: string, values: MessageValues, tags: string[]): ComponentChildren[] {
  const parts: ComponentChildren[] = [];
  let cursor = 0;

  for (const match of message.matchAll(pairMatcher(tags, 'g'))) {
    const start = match.index as number;

    if (start > cursor) {
      parts.push(interpolate(message.slice(cursor, start), values));
    }

    parts.push((values[match[1]] as ChunkFormatter)(interpolate(match[2], values)));
    cursor = start + match[0].length;
  }

  if (cursor < message.length) {
    parts.push(interpolate(message.slice(cursor), values));
  }

  return parts;
}

function format(message: string, fallback: string | undefined, values?: MessageValues): ComponentChildren {
  if (!values) {
    return message;
  }

  const tags = Object.keys(values).filter((key) => typeof values[key] === 'function');
  // a message that cannot be parsed falls back to the source instead of reaching the
  // page. this has to run whether or not the message carries rich text, since a mistyped
  // placeholder is the likelier mistake
  const source = isBroken(message, values, tags) && fallback !== undefined ? fallback : message;

  return tags.length === 0 ? interpolate(source, values) : formatRich(source, values, tags);
}

/**
 * `Intl.DateTimeFormat` is expensive to construct and every comment formats a date and a
 * time, so instances are reused.
 */
const formatters = new Map<string, Intl.DateTimeFormat>();

function dateTimeFormat(locale: string, options?: Intl.DateTimeFormatOptions): Intl.DateTimeFormat {
  const key = `${locale}\u0000${options ? JSON.stringify(options) : ''}`;
  let formatter = formatters.get(key);

  if (!formatter) {
    formatter = new Intl.DateTimeFormat(locale, options);
    formatters.set(key, formatter);
  }

  return formatter;
}

/** Formats a date, degrading to the raw value instead of throwing mid-render. */
function formatWith(locale: string, value: Date | number, options?: Intl.DateTimeFormatOptions): string {
  try {
    return dateTimeFormat(locale, options).format(value);
  } catch {
    return String(value);
  }
}

export function createIntl(locale: string, messages: Record<string, string>): IntlShape {
  function formatMessage(descriptor: MessageDescriptor, values?: MessageValues) {
    // the catalogue wins, then the message in the source. english ships an empty
    // catalogue, so defaultMessage is the primary path instead of a fallback.
    // an empty entry counts as missing
    const message = messages[descriptor.id] || descriptor.defaultMessage || descriptor.id;

    return format(message, descriptor.defaultMessage, values);
  }

  return {
    formatMessage: formatMessage as IntlShape['formatMessage'],
    locale,
    formatDate(value, options) {
      return formatWith(locale, value, options);
    },
    formatTime(value, options) {
      // hour and minute are added unless the caller asked for a time component itself;
      // every call here passes no options at all
      const needsDefaults =
        !options?.hour && !options?.minute && !options?.second && !options?.dateStyle && !options?.timeStyle;
      const resolved: Intl.DateTimeFormatOptions = needsDefaults
        ? { ...options, hour: 'numeric', minute: 'numeric' }
        : options;

      return formatWith(locale, value, resolved);
    },
  };
}
