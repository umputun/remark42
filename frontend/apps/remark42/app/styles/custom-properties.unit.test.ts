import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { describe, it } from 'node:test';

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8').replace(/\/\*[\s\S]*?\*\//g, '');

// the declarations of each top-level rule of a stylesheet without nesting, by its selector
function blocks(css: string): Map<string, Map<string, string>> {
  const found = new Map<string, Map<string, string>>();
  for (const [, selector, body] of css.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const declarations = new Map<string, string>();
    for (const [, name, value] of body.matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)) {
      declarations.set(name, value.trim());
    }
    found.set(selector.replace(/\s+/g, ' ').trim(), declarations);
  }
  return found;
}

const widget = blocks(source('./custom-properties.css'));
const lastComments = blocks(source('./last-comments-properties.css'));

// what last-comments.js bundles through components/list-comments/list-comments.js
const lastCommentsStyles = [
  '../components/comment/comment.module.css',
  '../components/comment/raw-content.css',
  '../components/list-comments/list-comments.module.css',
];

describe('the palette of last comments', () => {
  for (const [own, from] of [
    ['.remark42__last-comments', ':root'],
    ['.dark .remark42__last-comments, .remark42__last-comments.dark', ':root .dark'],
  ]) {
    it(`gives ${own} the values ${from} has in custom-properties.css`, () => {
      const declared = lastComments.get(own);
      const defaults = widget.get(from);
      assert.ok(declared, `last-comments-properties.css has no rule for ${own}`);
      assert.ok(defaults, `custom-properties.css has no rule for ${from}`);
      for (const [name, value] of declared) {
        assert.equal(value, defaults.get(name), `${name} in ${own}`);
      }
    });
  }

  it('declares every variable its stylesheets read', () => {
    const declared = lastComments.get('.remark42__last-comments');
    assert.ok(declared);
    for (const path of lastCommentsStyles) {
      for (const [, name] of source(path).matchAll(/var\((--[\w-]+)/g)) {
        assert.ok(declared.has(name), `${path} reads ${name}, which last comments does not declare`);
      }
    }
  });

  it('sets every variable the dark theme of custom-properties.css sets that its stylesheets read', () => {
    const dark = lastComments.get('.dark .remark42__last-comments, .remark42__last-comments.dark');
    const read = new Set(
      lastCommentsStyles.flatMap((path) => [...source(path).matchAll(/var\((--[\w-]+)/g)].map((m) => m[1]))
    );
    assert.ok(dark);
    for (const name of widget.get(':root .dark')?.keys() ?? []) {
      if (read.has(name)) {
        assert.ok(
          dark.has(name),
          `the dark theme sets ${name}, which last comments reads and does not set in the dark`
        );
      }
    }
  });
});
