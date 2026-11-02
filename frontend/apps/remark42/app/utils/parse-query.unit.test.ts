import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { parseQuery } from './parse-query.ts';

/**
 * parseQuery reads the query string the widget was embedded with, and every option the host page
 * sets reaches the code through it. The default argument is `window.location.search`, but a caller
 * that passes a string never evaluates it, which is what lets this run with no browser.
 */
describe('parseQuery', () => {
  it('returns an empty object for an empty search', () => {
    assert.deepEqual(parseQuery(''), {});
    assert.deepEqual(parseQuery('?'), {});
  });

  it('reads a valueless parameter as an empty string', () => {
    assert.deepEqual(parseQuery('?a'), { a: '' });
  });

  it('reads valueless and valued parameters side by side', () => {
    assert.deepEqual(parseQuery('?a&b=1'), { a: '', b: '1' });
  });

  it('reads every parameter', () => {
    assert.deepEqual(parseQuery('?a=1&b=1'), { a: '1', b: '1' });
  });

  it('decodes a percent-encoded value', () => {
    assert.deepEqual(parseQuery('?x=%D1%8B%D1%84%D0%B2%D0%B0%D1%84%D1%8B%D0%B2%D1%84%D1%8B'), { x: 'ыфвафывфы' });
  });

  it('keeps the last value when a parameter repeats', () => {
    assert.deepEqual(parseQuery('?a=1&a=2'), { a: '2' });
  });

  it('decodes a plus as a space', () => {
    assert.deepEqual(parseQuery('?a=one+two'), { a: 'one two' });
  });
});
