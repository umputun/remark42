import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { describe, it } from 'node:test';

import { searchEmoji } from './emoji-search.ts';

// the table the widget fetches, read from where the server embeds it, so the cases below hold
// for the file that ships and fail if it is replaced by something else
const table: Record<string, string> = JSON.parse(
  readFileSync(new URL('../../../../../../backend/app/webassets/assets/emoji.json', import.meta.url), 'utf8')
);

const names = (text: string) => searchEmoji(table, text).map(({ key }) => key);

describe('searchEmoji', () => {
  it('reads a table of the expected size', () => {
    assert.equal(Object.keys(table).length, 1878);
  });

  it('returns the names starting with the text, in the order of the table', () => {
    assert.deepEqual(searchEmoji(table, 'smi').slice(0, 5), [
      { key: 'smiley', emoji: '😃' },
      { key: 'smile', emoji: '😄' },
      { key: 'smiling_imp', emoji: '😈' },
      { key: 'smirk', emoji: '😏' },
      { key: 'smile_cat', emoji: '😸' },
    ]);
    assert.deepEqual(names('heart').slice(0, 5), [
      'heartbeat',
      'heartpulse',
      'heart_decoration',
      'heart_eyes',
      'heart_eyes_cat',
    ]);
    assert.deepEqual(names('flag-n').slice(0, 3), ['flag-na', 'flag-nc', 'flag-ne']);
  });

  // a name that only contains the text is not a match: "kissing_smiling_eyes" has "smi" in it
  it('matches at the start of a name only', () => {
    assert.ok('kissing_smiling_eyes' in table);
    assert.equal(names('smi').includes('kissing_smiling_eyes'), false);
  });

  it('finds a name that is not a word', () => {
    assert.deepEqual(searchEmoji(table, '+1'), [{ key: '+1', emoji: '👍' }]);
    assert.deepEqual(searchEmoji(table, '100'), [{ key: '100', emoji: '💯' }]);
  });

  it('returns nothing for a name the table does not have', () => {
    assert.deepEqual(searchEmoji(table, 'zzzz'), []);
  });

  it('is case sensitive, as the names are lower case', () => {
    assert.deepEqual(searchEmoji(table, 'SMI'), []);
  });

  // nothing typed after the colon yet: every name matches and the caller shows the first few
  it('returns the whole table for an empty text', () => {
    assert.equal(searchEmoji(table, '').length, 1878);
    assert.deepEqual(names('').slice(0, 5), ['100', '1234', 'umbrella_with_rain_drops', 'coffee', 'aries']);
  });

  it('ignores a closing colon', () => {
    assert.deepEqual(names('smile:'), ['smiley', 'smile', 'smile_cat', 'smiley_cat']);
  });

  it('searches for what follows a colon inside the text', () => {
    assert.deepEqual(names(':sm').slice(0, 5), [
      'small_orange_diamond',
      'small_blue_diamond',
      'small_red_triangle',
      'small_red_triangle_down',
      'smiley',
    ]);
  });
});
