import { test } from 'node:test';
import assert from 'node:assert/strict';

import { createIntl, defineMessages } from './intl-message.ts';

/**
 * The message formatter, tested as a parser over strings.
 *
 * A translator can introduce an unclosed tag, a tag carrying attributes or a brace that is not a
 * placeholder, and surviving those is what this code is for. Rendering a locale in a browser proves
 * the catalogue loads; it says nothing about what happens to a catalogue like that.
 *
 * `intl.test.tsx` covers much of the same ground under jest. What this file adds is that it runs
 * with no bundler, no jsdom and no `node_modules`: `node --test` and `bun test` both take it
 * unchanged, straight from source.
 */

const bold = (chunk: string) => ({ tag: 'b', chunk });

test('prefers the catalogue over the default message', () => {
  const intl = createIntl('en', { greeting: 'from the catalogue' });

  assert.equal(intl.formatMessage({ id: 'greeting', defaultMessage: 'from the source' }), 'from the catalogue');
});

test('falls back to the default message when the key is absent', () => {
  const intl = createIntl('en', {});

  assert.equal(intl.formatMessage({ id: 'greeting', defaultMessage: 'from the source' }), 'from the source');
});

test('treats an empty catalogue entry as missing', () => {
  const intl = createIntl('en', { greeting: '' });

  assert.equal(intl.formatMessage({ id: 'greeting', defaultMessage: 'from the source' }), 'from the source');
});

test('falls back to the id when there is no translation and no default', () => {
  const intl = createIntl('en', {});

  assert.equal(intl.formatMessage({ id: 'greeting' }), 'greeting');
});

test('interpolates a placeholder', () => {
  const intl = createIntl('en', {});

  assert.equal(intl.formatMessage({ id: 'x', defaultMessage: 'hello {name}' }, { name: 'world' }), 'hello world');
});

test('accepts whitespace inside the braces, as ICU does', () => {
  const intl = createIntl('en', {});

  assert.equal(intl.formatMessage({ id: 'x', defaultMessage: 'hello {  name  }' }, { name: 'world' }), 'hello world');
});

test('leaves a placeholder without a value untouched', () => {
  const intl = createIntl('en', {});

  // no values at all is the one path that formats the message as written
  assert.equal(intl.formatMessage({ id: 'x', defaultMessage: 'hello {name}' }), 'hello {name}');
});

test('falls back when a placeholder names a value the caller did not supply', () => {
  const intl = createIntl('en', { x: 'translated {missing}' });

  assert.equal(intl.formatMessage({ id: 'x', defaultMessage: 'source {name}' }, { name: 'world' }), 'source world');
});

test('falls back when a brace is not a well-formed placeholder', () => {
  const intl = createIntl('en', { x: 'translated {' });

  assert.equal(intl.formatMessage({ id: 'x', defaultMessage: 'source {name}' }, { name: 'world' }), 'source world');
});

test('does not interpolate a placeholder whose value is a handler', () => {
  const intl = createIntl('en', {});
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'a {b} c' }, { b: bold }) as unknown[];

  // the handler names a tag, not a value, so the braces stay text
  assert.deepEqual(out, ['a {b} c']);
});

test('wraps the tagged chunk with the handler', () => {
  const intl = createIntl('en', {});
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'a <b>bold</b> c' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['a ', { tag: 'b', chunk: 'bold' }, ' c']);
});

test('uses the translated text around the tag', () => {
  const intl = createIntl('en', { x: 'до <b>жирный</b> после' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'a <b>bold</b> c' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['до ', { tag: 'b', chunk: 'жирный' }, ' после']);
});

test('interpolates inside a rich-text chunk', () => {
  const intl = createIntl('en', {});
  const out = intl.formatMessage(
    { id: 'x', defaultMessage: '<b>hi {name}</b>' },
    { b: bold, name: 'you' }
  ) as unknown[];

  assert.deepEqual(out, [{ tag: 'b', chunk: 'hi you' }]);
});

test('handles the same tag appearing more than once', () => {
  const intl = createIntl('en', {});
  const out = intl.formatMessage({ id: 'x', defaultMessage: '<b>one</b> and <b>two</b>' }, { b: bold }) as unknown[];

  assert.deepEqual(out, [{ tag: 'b', chunk: 'one' }, ' and ', { tag: 'b', chunk: 'two' }]);
});

test('renders a translation that does not use the tag at all', () => {
  const intl = createIntl('en', { x: 'no markup here' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: '<b>bold</b>' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['no markup here']);
});

test('falls back to the default message when a tag is unclosed', () => {
  const intl = createIntl('en', { x: 'broken <b>bold' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'a <b>bold</b> c' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['a ', { tag: 'b', chunk: 'bold' }, ' c']);
});

test('falls back when a tag carries attributes', () => {
  const intl = createIntl('en', { x: 'a <b class="x">bold</b> c' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'a <b>bold</b> c' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['a ', { tag: 'b', chunk: 'bold' }, ' c']);
});

test('falls back when the translation introduces a tag nothing handles', () => {
  const intl = createIntl('en', { x: 'a <i>italic</i> c' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'a <b>bold</b> c' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['a ', { tag: 'b', chunk: 'bold' }, ' c']);
});

test('falls back when a stray tag sits beside a well-formed pair', () => {
  const intl = createIntl('en', { x: '<b>bold</b> and </i>' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'a <b>bold</b> c' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['a ', { tag: 'b', chunk: 'bold' }, ' c']);
});

test('falls back when a handled tag is nested inside another', () => {
  const intl = createIntl('en', { x: '<b>outer <b>inner</b></b>' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'a <b>bold</b> c' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['a ', { tag: 'b', chunk: 'bold' }, ' c']);
});

test('falls back when an unhandled tag is nested inside a handled pair', () => {
  // the stray closing tag of a doubled pair is caught by the residue check alone; only a tag left
  // whole inside a chunk reaches the chunk check, and without it `<i>x</i>` would be handed to the
  // handler and reach the page as raw text
  const intl = createIntl('en', { x: '<b>outer <i>x</i></b>' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'a <b>bold</b> c' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['a ', { tag: 'b', chunk: 'bold' }, ' c']);
});

test('renders a self-closing tag as text', () => {
  // the catalogue entry has to differ from the default message, or falling back and not falling
  // back produce the same string and the verdict is unobservable
  const intl = createIntl('en', { x: 'line<br />break' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'the default' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['line<br />break']);
});

test('leaves a bare less-than sign alone', () => {
  const intl = createIntl('en', { x: '5 < 10' });
  const out = intl.formatMessage({ id: 'x', defaultMessage: 'the default' }, { b: bold }) as unknown[];

  assert.deepEqual(out, ['5 < 10']);
});

test('renders the broken message when there is nothing to fall back to', () => {
  const intl = createIntl('en', { x: 'broken <b>bold' });
  const out = intl.formatMessage({ id: 'x' }, { b: bold }) as unknown[];

  // no defaultMessage, so the catalogue entry is all there is
  assert.deepEqual(out, ['broken <b>bold']);
});

test('escapes a tag name that carries a regular-expression metacharacter', () => {
  const intl = createIntl('en', {});
  const dot = (chunk: string) => ({ tag: 'a.b', chunk });
  // the decoy has to sit first and match the unescaped pattern, or `.` matching the literal `.`
  // gives the same answer as escaping it and the case passes against the defect it names
  const out = intl.formatMessage(
    { id: 'x', defaultMessage: '<axb>decoy</axb><a.b>text</a.b>' },
    { 'a.b': dot }
  ) as unknown[];

  // unescaped, the pattern for `a.b` matches the decoy first and hands it to the handler
  assert.deepEqual(out, ['<axb>decoy</axb>', { tag: 'a.b', chunk: 'text' }]);
});

test('builds a usable pattern from a tag name that would be an invalid regular expression', () => {
  const intl = createIntl('en', {});
  const paren = (chunk: string) => ({ tag: 'a(b', chunk });

  // unescaped, `new RegExp` throws on the unbalanced parenthesis and takes the render down
  assert.deepEqual(intl.formatMessage({ id: 'x', defaultMessage: '<a(b>text</a(b>' }, { 'a(b': paren }), [
    { tag: 'a(b', chunk: 'text' },
  ]);
});

test('defineMessages returns its argument unchanged', () => {
  const messages = { a: { id: 'a', defaultMessage: 'A' } };

  assert.equal(defineMessages(messages), messages);
});

test('formats time with hour and minute by default', () => {
  const intl = createIntl('en-GB', {});
  const at = Date.UTC(2020, 0, 2, 3, 4, 5);

  // the value is formatted in the runtime's zone, so the assertion is on the shape instead of on
  // a wall-clock reading: two numbers separated by a colon, and no seconds
  assert.match(intl.formatTime(at), /^\d{2}:\d{2}$/);
});

// each of these is an independent reason not to add the hour and minute defaults, so one case
// cannot stand for the rest: a guard that checks only some of them still passes the others
test('honours explicit time options instead of adding defaults', () => {
  const intl = createIntl('en-GB', {});
  const at = Date.UTC(2020, 0, 2, 3, 4, 5);

  for (const options of [
    { hour: 'numeric' },
    { minute: 'numeric' },
    { second: 'numeric' },
  ] as Intl.DateTimeFormatOptions[]) {
    assert.match(intl.formatTime(at, options), /^\d{1,2}$/, JSON.stringify(options));
  }

  // a style asks for a whole preset, and Intl rejects an hour or a minute beside one, so adding the
  // defaults there makes the format throw and the value degrade to its raw self. the assertion is
  // on the rendered shape, which is what tells a real format from that fallback
  assert.match(intl.formatTime(at, { timeStyle: 'medium' }), /^\d{2}:\d{2}:\d{2}$/);
  assert.match(intl.formatTime(at, { dateStyle: 'short' }), /^\d{2}\/\d{2}\/\d{4}$/);
});

test('does not confuse cached formatters across locales', () => {
  const at = Date.UTC(2026, 0, 2, 15, 4);

  assert.notEqual(createIntl('ru', {}).formatDate(at), createIntl('en', {}).formatDate(at));
  assert.equal(createIntl('en', {}).formatDate(at), createIntl('en', {}).formatDate(at));
});

test('falls back when the translation adds a pair nothing handles beside one it does', () => {
  const intl = createIntl('en', { k: 'See <abbr>HTML</abbr> and <a>link</a>' });
  const out = intl.formatMessage({ id: 'k', defaultMessage: 'Go <a>home</a>' }, { a: (c: string) => c });

  assert.deepEqual(out, ['Go ', 'home']);
});

// the tag name is matched to its end, so a name that merely starts with a handled one is a
// different tag, and a translation carrying it falls back
test('treats a non-ascii suffix as part of the tag name, and falls back', () => {
  const intl = createIntl('en', { k: '<a\u0431>X</a\u0431>' });
  const out = intl.formatMessage({ id: 'k', defaultMessage: 'Go <a>home</a>' }, { a: (c: string) => c });

  assert.deepEqual(out, ['Go ', 'home']);
});

test('does not confuse cached formatters across options of one locale', () => {
  const at = Date.UTC(2020, 0, 2, 3, 4, 5);
  const intl = createIntl('en-GB', {});

  // one locale, two option sets: a cache keyed on locale alone hands the second read the formatter
  // built for the first, and the two come back identical. two locales would not show it, since a
  // locale-only key still tells those apart
  const short = intl.formatDate(at, { dateStyle: 'short' });
  const full = intl.formatDate(at, { dateStyle: 'full' });

  assert.notEqual(short, full);
});

test('degrades to the raw value when a date cannot be formatted', () => {
  const intl = createIntl('en', {});

  // an out-of-range option throws inside Intl, and a widget mid-render must not
  assert.equal(intl.formatDate(0, { timeZone: 'Not/AZone' }), '0');
});
