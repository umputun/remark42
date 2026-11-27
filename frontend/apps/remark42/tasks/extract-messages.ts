import { mkdirSync, readdirSync, readFileSync, realpathSync, writeFileSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { pathToFileURL } from 'node:url';

export type Source = { path: string; text: string };
export type Extraction = { messages: Record<string, { defaultMessage: string }>; errors: string[] };

// a quoted string in any of the three quote styles, with escapes, captured as `<name>` with its quote as
// `<name>Q`
const string = (name: string) =>
  String.raw`(?<${name}Q>['"\x60])(?<${name}>(?:\\[\s\S]|(?!\k<${name}Q>)[^\\])*)\k<${name}Q>`;

// a key, bare or quoted, and the separator after it: `:` in an object literal, `=` for a JSX attribute,
// whose value may be wrapped in braces. The spacing is prettier's, which CI enforces: nothing before `:`,
// and nothing on either side of a JSX `=`. That is what tells a key from a ternary (`a ? key : 'x'`), an
// assignment (`key = 'x'`) or a destructuring default (`{ key = 'x' }`), none of which is a message
const key = (name: string, label: string) =>
  String.raw`(?<![\w$.])(?<${label}K>['"]?)${name}\k<${label}K>(?<${label}Sep>:|=(?=[^\s=]))\s*(?<${label}Brace>\{?)\s*`;

// anything a comment can put between a string and what follows it
const GAP = String.raw`(?:\s|\/\*[\s\S]*?\*\/|\/\/[^\n]*\n)*`;

// a message is an `id` followed by its `defaultMessage`, in an object literal (`id: 'a', defaultMessage: 'b'`)
// or as JSX attributes (`id="a" defaultMessage="b"`). What is around them does not matter, so a helper
// renamed or a message moved to another call shape is still found
const PAIR = new RegExp(
  key('id', 'id') +
    string('id') +
    String.raw`\s*\}?\s*,?\s*` +
    key('defaultMessage', 'msg') +
    string('msg') +
    // a text that goes on past its closing quote (`'a' + b`, `'a'.trim()`) is not a plain string, and reading
    // its first part as the whole would put a truncated message in the catalogue
    String.raw`(?!${GAP}[-+*.[(|&])`,
  'g'
);

// every place a message text is given as a string: the key, then its value starting with a quote. A
// concatenation or a call on that string is counted and not read, so it fails; a union of string types
// (`key: 'a' | 'b'`) is not a message. Reading, typing or destructuring the property does not match, and
// a text given by a variable is not seen at all: if an existing message disappears that way, the check
// against en.json in main stops the run
const DEFINITION = new RegExp(
  key('defaultMessage', 'def') + string('def') + String.raw`(?!${GAP}(?:\|(?!\|)|&(?!&)))`,
  'g'
);

/**
 * The source as the reader sees it: line endings as JavaScript normalises them. Comments are read like code,
 * so a commented-out message keeps its key in the catalogues; the reader errs towards keeping a key, never
 * towards losing one.
 */
export function readable(text: string): string {
  return text.replace(/\r\n?/g, '\n');
}

const SKIP = [/\.d\.ts$/, /\.(test|spec)\.[jt]sx?$/, /(^|\/)tests\//, /(^|\/)__mocks__\//, /(^|\/)__stubs__\//];

const SIMPLE_ESCAPES: Record<string, string> = { b: '\b', f: '\f', n: '\n', r: '\r', t: '\t', v: '\v', 0: '\0' };

/** The value of a JavaScript string literal's body, escapes resolved as the language resolves them. */
export function unescapeString(body: string): string {
  return body.replace(
    /\\(?:x([0-9a-fA-F]{2})|u\{([0-9a-fA-F]+)\}|u([0-9a-fA-F]{4})|(\r\n|[\n\r\u2028\u2029])|([\s\S]))/g,
    (_, hex: string, codePoint: string, unit: string, lineBreak: string, char: string) => {
      if (hex) return String.fromCharCode(parseInt(hex, 16));
      if (codePoint) return String.fromCodePoint(parseInt(codePoint, 16));
      if (unit) return String.fromCharCode(parseInt(unit, 16));
      if (lineBreak) return '';
      return SIMPLE_ESCAPES[char] ?? char;
    }
  );
}

// a JSX attribute written as `attr="..."` takes its text as written, backslashes included, turns `&...;`
// into characters and folds a line break with the indent after it into one space. That is not reproduced
// here, so such a text is refused instead of guessed
function unreadableAttribute(separator: string, brace: string, body: string): boolean {
  return separator === '=' && !brace && /[&\\\n]/.test(body);
}

/** Finds every message in the sources, and reports each definition it could not read. */
export function extractMessages(sources: Source[]): Extraction {
  const messages: Record<string, { defaultMessage: string }> = {};
  const errors: string[] = [];

  for (const source of sources) {
    const { path } = source;
    const text = readable(source.text);
    let found = 0;

    for (const match of text.matchAll(PAIR)) {
      found++;
      const g = match.groups!;

      if ((g.idQ === '`' && g.id.includes('${')) || (g.msgQ === '`' && g.msg.includes('${'))) {
        errors.push(`${path}: "${g.id}" is built from a template expression, so its text is not known here`);
        continue;
      }

      if (unreadableAttribute(g.idSep, g.idBrace, g.id) || unreadableAttribute(g.msgSep, g.msgBrace, g.msg)) {
        errors.push(
          `${path}: "${g.id}" is a JSX attribute with a backslash, an entity or a line break; write it as {'...'}`
        );
        continue;
      }

      // an attribute that reached this point has no backslash, so resolving escapes leaves it as written
      const id = unescapeString(g.id);
      const defaultMessage = unescapeString(g.msg);
      const known = messages[id];

      if (known && known.defaultMessage !== defaultMessage) {
        errors.push(
          `${path}: "${id}" is defined again with different text: ` +
            `${JSON.stringify(defaultMessage)} against ${JSON.stringify(known.defaultMessage)}`
        );
        continue;
      }
      messages[id] = { defaultMessage };
    }

    const defined = text.match(DEFINITION)?.length ?? 0;

    if (defined !== found) {
      errors.push(
        `${path}: ${defined} message definitions but ${found} could be read; each needs an \`id\` written ` +
          `just before its \`defaultMessage\`, both as plain strings`
      );
    }
  }

  const sorted = Object.fromEntries(Object.entries(messages).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0)));

  return { messages: sorted, errors };
}

/** Keys the English catalogue holds that the extraction no longer finds. */
export function droppedKeys(messages: Record<string, unknown>, english: Record<string, unknown>): string[] {
  return Object.keys(english).filter((key) => !(key in messages));
}

export function sourcesIn(root: string): Source[] {
  return readdirSync(root, { recursive: true, encoding: 'utf8' })
    .map((file) => join(root, file))
    .filter((file) => /\.(js|tsx?)$/.test(file) && !SKIP.some((re) => re.test(relative(root, file))))
    .sort()
    .map((file) => ({ path: relative(root, file), text: readFileSync(file, 'utf8') }));
}

function main(): void {
  const app = join(import.meta.dirname, '..');
  const out = join(app, 'extracted-messages', 'messages.json');
  const { messages, errors } = extractMessages(sourcesIn(join(app, 'app')));

  if (errors.length) {
    console.error(errors.join('\n'));
    process.exit(1);
  }

  // translation:generate deletes from every catalogue whatever is missing here, so a key that goes missing
  // is stopped until the removal is confirmed rather than taken as one
  const english = JSON.parse(
    readFileSync(join(app, '..', '..', '..', 'backend', 'app', 'webassets', 'assets', 'en.json'), 'utf8')
  );
  const dropped = droppedKeys(messages, english);

  if (dropped.length && !process.argv.includes('--allow-removal')) {
    console.error(
      `${dropped.length} of the ${Object.keys(english).length} messages in en.json are no longer in the source: ` +
        `${dropped.join(', ')}.\nIf they were removed on purpose, run "pnpm translation:extract --allow-removal" ` +
        `and then "pnpm translation:generate".`
    );
    process.exit(1);
  }

  mkdirSync(dirname(out), { recursive: true });
  writeFileSync(out, `${JSON.stringify(messages, null, 2)}\n`);
  console.log(`${Object.keys(messages).length} messages written to ${relative(app, out)}`);
}

// run as a script, not when a test imports it. import.meta.main would say so, but only from node 24.2
if (process.argv[1] && import.meta.url === pathToFileURL(realpathSync(process.argv[1])).href) {
  main();
}
