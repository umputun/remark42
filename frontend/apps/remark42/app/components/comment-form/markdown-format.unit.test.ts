import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { formatSelection, STYLES } from './markdown-format.js';

/**
 * The text after a button, with the selection marked: `[` and `]` around a selection, `|` for a
 * caret. The input is written the same way.
 */
function apply(marked: string, style: keyof typeof STYLES): string {
  const start = marked.search(/[[|]/);
  const end = marked.includes('|') ? start : marked.indexOf(']') - 1;
  const value = marked.replace(/[[\]|]/g, '');
  const edit = formatSelection(value, start, end, STYLES[style]);
  const text = value.slice(0, edit.start) + edit.text + value.slice(edit.end);
  return edit.selectionStart === edit.selectionEnd
    ? `${text.slice(0, edit.selectionStart)}|${text.slice(edit.selectionStart)}`
    : `${text.slice(0, edit.selectionStart)}[${text.slice(edit.selectionStart, edit.selectionEnd)}]${text.slice(edit.selectionEnd)}`;
}

describe('formatSelection', () => {
  it('heads the line with a third-level heading at the caret', () => {
    assert.equal(apply('wo|rd', 'header'), '### wo|rd');
    assert.equal(apply('|', 'header'), '### |');
  });

  it('wraps the selection, or the word at the caret, in bold and italic markers', () => {
    assert.equal(apply('[word]', 'bold'), '**[word]**');
    assert.equal(apply('wo|rd', 'bold'), '**wo|rd**');
    assert.equal(apply('[hello] world', 'italic'), '_[hello]_ world');
  });

  it('keeps the whitespace at the edges of a selection outside the markers', () => {
    assert.equal(apply('[  padded  ]', 'bold'), '  **[padded]**  ');
  });

  it('takes the markers off a selection that has them around it', () => {
    assert.equal(apply('a **[b]** c', 'bold'), 'a [b] c');
    assert.equal(apply('a [**b**] c', 'bold'), 'a [b] c');
    assert.equal(apply('a _[b]_ c', 'italic'), 'a [b] c');
  });

  it('fences a selection of several lines as a block of code', () => {
    assert.equal(apply('[two\nlines]', 'code'), '```\n[two\nlines]\n```');
    assert.equal(apply('[one line]', 'code'), '`[one line]`');
  });

  it('quotes in a paragraph of its own, line by line, and unquotes', () => {
    assert.equal(apply('before [middle] after', 'quote'), 'before \n\n> [middle]\n\n after');
    assert.equal(apply('[two\nlines]', 'quote'), '[> two\n> lines]');
    assert.equal(apply('[> one\n> two]', 'quote'), '[one\ntwo]');
  });

  it('makes a link of the text with the address selected, and of an address with the text selected', () => {
    assert.equal(apply('a [link]', 'link'), 'a [link]([url])');
    assert.equal(apply('[https://example.org]', 'link'), '[|](https://example.org)');
    assert.equal(apply('|', 'link'), '[|](url)');
  });

  it('makes lists of whole lines, swaps one kind for the other and takes them off', () => {
    assert.equal(apply('one\n[two\nthree]', 'unordered-list'), 'one\n\n[- two\n- three]');
    assert.equal(apply('[one\ntwo]', 'ordered-list'), '[1. one\n2. two]');
    assert.equal(apply('[- a\n- b]', 'ordered-list'), '[1. a\n2. b]');
    assert.equal(apply('[- a\n- b]', 'unordered-list'), '[a\nb]');
    assert.equal(apply('1. |a\n2. b', 'ordered-list'), '|a\n2. b');
    assert.equal(apply('fi|rst', 'unordered-list'), '- fi|rst');
  });
});
