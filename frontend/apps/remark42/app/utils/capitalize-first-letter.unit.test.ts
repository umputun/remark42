import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { capitalizeFirstLetter } from './capitalize-first-letter.ts';

describe('capitalizeFirstLetter', () => {
  it('raises the first letter and leaves the rest alone', () => {
    assert.equal(capitalizeFirstLetter('one'), 'One');
    assert.equal(capitalizeFirstLetter('one two'), 'One two');
  });

  it('uppercases outside ASCII', () => {
    assert.equal(capitalizeFirstLetter('один'), 'Один');
  });

  // a script with no case distinction must come back untouched, since the widget applies this to
  // every locale's strings
  it('leaves a caseless script unchanged', () => {
    assert.equal(capitalizeFirstLetter('用户名最少需要3个字符'), '用户名最少需要3个字符');
  });

  it('is a no-op on something already capitalised', () => {
    assert.equal(capitalizeFirstLetter('One'), 'One');
  });

  it('returns an empty string for an empty string', () => {
    assert.equal(capitalizeFirstLetter(''), '');
  });

  // uppercasing is not always length preserving: ß maps to SS, so a caller measuring the result
  // against the input cannot assume the two match
  it('keeps the expansion when uppercasing lengthens the character', () => {
    assert.equal(capitalizeFirstLetter('ßeta'), 'SSeta');
  });

  it('does not split a surrogate pair', () => {
    assert.equal(capitalizeFirstLetter('😀ok'), '😀ok');
  });
});
