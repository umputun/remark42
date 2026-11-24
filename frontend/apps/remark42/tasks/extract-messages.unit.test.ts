import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { describe, it } from 'node:test';

import { droppedKeys, extractMessages, sourcesIn } from './extract-messages.ts';

const one = (text: string) => extractMessages([{ path: 'a.tsx', text }]);

describe('extractMessages', () => {
  it('reads an object literal and a JSX element', () => {
    const { messages, errors } = one(`
      const m = defineMessages({ back: { id: 'auth.back', defaultMessage: 'Back' } });
      <FormattedMessage id="auth.or" defaultMessage="or" />
    `);

    assert.deepEqual(errors, []);
    assert.deepEqual(messages, { 'auth.back': { defaultMessage: 'Back' }, 'auth.or': { defaultMessage: 'or' } });
  });

  // the identifiers around a message are not what finds it, so renaming the helper that wraps it
  // cannot make a message disappear from the catalogue
  it('does not depend on the name of the function the message is passed to', () => {
    const { messages } = one(`const m = renamedHelper({ x: { id: 'x', defaultMessage: 'X' } });`);

    assert.deepEqual(messages, { x: { defaultMessage: 'X' } });
  });

  it('resolves escapes the way JavaScript does, in every quote style', () => {
    const { messages, errors } = one(
      [
        String.raw`{ id: "a", defaultMessage: "say \"hi\"" } { id: 'b', defaultMessage: 'it\'s' } `,
        String.raw`{ id: 'c', defaultMessage: 'tab\there, \u2026, \x41, \u{1F600}, back\\slash' } `,
        '{ id: `d`, defaultMessage: `line\\nnext` }',
      ].join(' ')
    );

    assert.deepEqual(errors, []);
    assert.deepEqual(messages, {
      a: { defaultMessage: 'say "hi"' },
      b: { defaultMessage: "it's" },
      c: { defaultMessage: 'tab\there, …, A, \u{1F600}, back\\slash' },
      d: { defaultMessage: 'line\nnext' },
    });
  });

  // in JSX, `attr="a\nb"` is a backslash and an n, and `attr="&amp;"` is an ampersand
  it('takes a JSX attribute as written, and refuses one with a backslash, an entity or a line break', () => {
    assert.deepEqual(one(`<FormattedMessage id="x" defaultMessage="Plain text" />`).messages, {
      x: { defaultMessage: 'Plain text' },
    });
    assert.deepEqual(one(`<FormattedMessage id="x" defaultMessage={'it\\'s'} />`).messages, {
      x: { defaultMessage: "it's" },
    });

    for (const text of [
      String.raw`<M id="x" defaultMessage="a\nb" />`,
      `<M id="x" defaultMessage="Q&amp;A" />`,
      `<M id="x" defaultMessage="first\n      second" />`,
    ]) {
      const { messages, errors } = one(text);

      assert.deepEqual(messages, {}, text);
      assert.match(errors[0], /a JSX attribute with a backslash, an entity or a line break/, text);
    }
  });

  it('reads quoted keys', () => {
    const { messages, errors } = one(
      `{ "id": "q.a", "defaultMessage": "Quoted" } { id: 'q.b', 'defaultMessage': 'Half' }`
    );

    assert.deepEqual(errors, []);
    assert.deepEqual(messages, { 'q.a': { defaultMessage: 'Quoted' }, 'q.b': { defaultMessage: 'Half' } });
  });

  it('sorts the keys', () => {
    const { messages } = one(`{ id: 'b', defaultMessage: 'B' } { id: 'a', defaultMessage: 'A' }`);

    assert.deepEqual(Object.keys(messages), ['a', 'b']);
  });

  it('accepts one id defined twice with the same text', () => {
    const { messages, errors } = extractMessages([
      { path: 'a.tsx', text: `{ id: 'token', defaultMessage: 'Token' }` },
      { path: 'b.tsx', text: `{ id: 'token', defaultMessage: 'Token' }` },
    ]);

    assert.deepEqual(errors, []);
    assert.deepEqual(messages, { token: { defaultMessage: 'Token' } });
  });

  // one of the two texts would be lost without anybody choosing which
  it('fails when one id is defined twice with different text', () => {
    const { errors } = extractMessages([
      { path: 'a.tsx', text: `{ id: 'token', defaultMessage: 'Token' }` },
      { path: 'b.tsx', text: `{ id: 'token', defaultMessage: 'Code' }` },
    ]);

    assert.equal(errors.length, 1);
    assert.match(errors[0], /^b\.tsx: "token" is defined again with different text/);
  });

  // a message it cannot read would vanish from the catalogue on the next generate, so it is reported
  it('fails on a message text it cannot read', () => {
    for (const text of [
      `{ id: 'a', defaultMessage: 'one ' + 'two' }`,
      `{ id: 'a', defaultMessage: 'Text'.trim() }`,
      `{ id: 'a', defaultMessage: 'Hello' /* note */ + ' world' }`,
      `{ id: 'a', defaultMessage: 'Text' || fallback }`,
      `{ id: 'a', defaultMessage: 'Text' && other }`,
      `{ defaultMessage: 'B', id: 'b' }`,
      `{ id: someVariable, defaultMessage: 'C' }`,
    ]) {
      const { errors } = one(text);

      assert.equal(errors.length, 1, text);
      assert.match(errors[0], /^a\.tsx: 1 message definitions but 0 could be read/, text);
    }
  });

  it('fails on a template literal with an expression in it', () => {
    // the expression is the text under test, not one to fill in here
    // eslint-disable-next-line no-template-curly-in-string
    const { messages, errors } = one('{ id: `a`, defaultMessage: `Hello ${name}` }');

    assert.deepEqual(messages, {});
    assert.equal(errors.length, 1);
    assert.match(errors[0], /built from a template expression/);
  });

  it('does not count reading, typing or destructuring the property as a definition', () => {
    const { messages, errors } = one(`
      type Descriptor = { id: string; defaultMessage?: string };
      type Required = { id: string; defaultMessage: string };
      throw new Error(descriptor.defaultMessage as string);
      function read({ defaultMessage }: { defaultMessage?: string }) { return format(defaultMessage, values); }
      const short = { id: 'a', defaultMessage };
      function withDefault({ defaultMessage = '' }: { defaultMessage?: string }) { return defaultMessage; }
      const text = known ? defaultMessage : 'Unknown';
      let defaultMessage = 'Unknown';
      type Kind = { id: string; defaultMessage: 'a' | 'b' };
      const { id = 'fallback.id', defaultMessage = 'Fallback' } = descriptor;
    `);

    assert.deepEqual(errors, []);
    assert.deepEqual(messages, {});
  });

  // reading a comment like code keeps a commented-out message's key; telling comments from code would take
  // a parser, and a guess at it once blanked the text of live strings and templates
  it('reads a message in a comment like code, and loses nothing next to a comment marker in a template', () => {
    const { messages, errors } = one(
      [
        `// const old = { id: 'gone', defaultMessage: 'Gone' };`,
        'const css = `',
        '/* start',
        "`; const b = { id: 'b', defaultMessage: 'B' }; // */",
        `const live = { id: 'live', defaultMessage: 'See https://remark42.com' };`,
      ].join('\n')
    );

    assert.deepEqual(errors, []);
    assert.deepEqual(Object.keys(messages), ['b', 'gone', 'live']);
  });

  it('reads a template literal written with Windows line endings as JavaScript does', () => {
    const { messages } = one('{ id: `crlf`, defaultMessage: `first\r\nsecond` }');

    assert.deepEqual(messages, { crlf: { defaultMessage: 'first\nsecond' } });
  });
});

describe('droppedKeys', () => {
  it('lists the catalogue keys the extraction no longer finds', () => {
    assert.deepEqual(droppedKeys({ a: {} }, { a: 'A', b: 'B' }), ['b']);
    assert.deepEqual(droppedKeys({ a: {}, b: {} }, { a: 'A' }), []);
  });
});

describe('the widget source', () => {
  // the whole tree, read the way translation:extract reads it: if a change to the code or to this
  // reader loses a message, the English catalogue says so
  it('yields exactly the messages of the English catalogue, with no errors', () => {
    const { messages, errors } = extractMessages(sourcesIn(new URL('../app', import.meta.url).pathname));
    const english = JSON.parse(
      readFileSync(new URL('../../../../backend/app/webassets/assets/en.json', import.meta.url), 'utf8')
    );

    assert.deepEqual(errors, []);
    assert.deepEqual(Object.keys(messages), Object.keys(english).sort());
    for (const [key, { defaultMessage }] of Object.entries(messages)) {
      assert.equal(defaultMessage, english[key], key);
    }
  });
});
