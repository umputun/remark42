/*!
 * Remark42's emoji completion is adapted from @github/text-expander-element 2.9.4 and
 * @github/combobox-nav 2.3.1, and the list of properties a copy of the field takes from
 * dom-input-range 2.0.1, under their licences:
 *
 * Copyright (c) 2019 GitHub, Inc. (text-expander-element)
 * Copyright (c) 2018 GitHub (combobox-nav)
 * Copyright (c) 2023 Ian Sanders (dom-input-range), with portions
 * Copyright (c) 2018 GitHub, Inc. and Copyright (c) 2015 Jonathan Ong me@jongleberry.com
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// The emoji completion of the comment field, in plain JavaScript. A colon followed by the start of
// an emoji name opens a list of up to five matches under the caret, as the text expander and
// combobox packages did: the first match is selected, the arrows move the selection, Enter or Tab
// or a click puts the emoji's `:name:` and a space in place of what was typed, and Escape or
// moving the focus away closes the list.

import { classes } from '../../common/dom.js';
import { searchEmoji } from './emoji-search.ts';
import { insertText } from './insert-text.js';
import styles from './text-expander.module.css';

const KEY = ':';
// a colon starts a name only at the start of the text or after one of these
const BOUNDARY = /\s|\(|\[/;
const LIMIT = 5;
// the properties that decide where a line of the field's text wraps and how tall it is
const MIRRORED = /** @type {const} */ ([
  'direction',
  'writingMode',
  'unicodeBidi',
  'textOrientation',
  'boxSizing',
  'borderTopWidth',
  'borderRightWidth',
  'borderBottomWidth',
  'borderLeftWidth',
  'borderStyle',
  'paddingTop',
  'paddingRight',
  'paddingBottom',
  'paddingLeft',
  'fontStyle',
  'fontVariant',
  'fontWeight',
  'fontStretch',
  'fontSize',
  'fontSizeAdjust',
  'lineHeight',
  'fontFamily',
  'textAlign',
  'textTransform',
  'textIndent',
  'textDecoration',
  'letterSpacing',
  'wordSpacing',
  'tabSize',
]);

/**
 * The name typed after the colon before the caret, and where it starts, when the text there is
 * one: the colon is the last one before the caret, no later than lookBackIndex is not too far
 * back, no space comes after it, and it starts the text or follows a space or bracket.
 *
 * @param {string} text
 * @param {number} cursor
 * @param {number} lookBackIndex  a colon before this has been dealt with
 * @returns {{ text: string, position: number } | undefined}
 */
export function findQuery(text, cursor, lookBackIndex) {
  const keyIndex = text.lastIndexOf(KEY, cursor - 1);
  if (keyIndex === -1 || keyIndex < lookBackIndex) {
    return undefined;
  }
  if (text.lastIndexOf(' ', cursor - 1) > keyIndex) {
    return undefined;
  }
  const before = text[keyIndex - 1];
  if (before && !BOUNDARY.test(before)) {
    return undefined;
  }
  return { text: text.substring(keyIndex + KEY.length, cursor), position: keyIndex + KEY.length };
}

/**
 * Where the caret would be drawn before this character of the field's text, in the page's
 * coordinates: a hidden copy of the field with the same box, font and text is laid out, and the
 * caret's place in it measured with a collapsed range, as dom-input-range measured it.
 *
 * @param {HTMLTextAreaElement} textarea
 * @param {number} position
 */
function caretRect(textarea, position) {
  const style = window.getComputedStyle(textarea);
  const mirror = document.createElement('div');
  mirror.setAttribute('aria-hidden', 'true');
  for (const property of MIRRORED) {
    mirror.style[property] = style[property];
  }
  Object.assign(mirror.style, {
    position: 'absolute',
    top: '0',
    left: '0',
    visibility: 'hidden',
    pointerEvents: 'none',
    overflow: 'hidden',
    whiteSpace: 'pre-wrap',
    wordWrap: 'break-word',
    width: style.width,
    height: style.height,
  });
  // the whole text, since what follows the position decides where its line wraps
  const text = document.createTextNode(textarea.value);
  mirror.append(text);
  textarea.after(mirror);
  const range = document.createRange();
  range.setStart(text, position);
  range.setEnd(text, position);
  const field = textarea.getBoundingClientRect();
  const box = mirror.getBoundingClientRect();
  const at = range.getBoundingClientRect();
  mirror.remove();
  return {
    left: field.left + at.left - box.left - textarea.scrollLeft,
    top: field.top + at.top - box.top - textarea.scrollTop,
    height: at.height,
  };
}

/**
 * Whether an option takes space on the page, which is what the combobox counted as visible.
 *
 * @param {HTMLElement} element
 */
function visible(element) {
  return !element.hidden && (element.offsetWidth > 0 || element.offsetHeight > 0);
}

/**
 * @typedef {object} EmojiCompletionProps
 * @property {import('../../common/types.ts').Theme} theme
 */

/**
 * @typedef {object} EmojiCompletionDeps
 * @property {() => Promise<Record<string, string>>} loadEmoji  the name-to-emoji table
 */

/**
 * Adds the completion to a field. The list is drawn inside host, which holds the field, and
 * positioned there under the caret.
 *
 * @param {HTMLElement} host
 * @param {HTMLTextAreaElement} textarea
 * @param {EmojiCompletionProps} initial
 * @param {EmojiCompletionDeps} initialDeps
 */
export function emojiCompletion(host, textarea, initial, initialDeps) {
  let props = initial;
  let deps = initialDeps;
  // the combobox reads Ctrl+N and Ctrl+P as the arrows on a Mac
  const ctrlBindings = Boolean(navigator.userAgent.match(/Macintosh/));

  /** @type {{ text: string, position: number } | null} */
  let match = null;
  /** @type {HTMLUListElement | null} */
  let menu = null;
  let lookBackIndex = 0;
  let justPasted = false;
  let interactingWithList = false;
  let composing = false;
  // set while the completion puts an emoji in, so the input that causes is not read as typing
  let inserting = false;

  /**
   * The list for a name, or null when nothing matches or the table cannot be had.
   *
   * @param {string} text
   * @param {import('../../common/types.ts').Theme} theme
   * @returns {Promise<HTMLUListElement | null>}
   */
  async function suggestions(text, theme) {
    let found;
    try {
      found = searchEmoji(await deps.loadEmoji(), text);
    } catch {
      return null;
    }
    if (found.length === 0) {
      return null;
    }
    const dark = theme === 'dark';
    const list = document.createElement('ul');
    list.className = classes(styles.suggester, dark && styles.suggesterDark);
    for (const { key, emoji } of found.slice(0, LIMIT)) {
      const item = document.createElement('li');
      item.setAttribute('role', 'option');
      item.className = classes(styles.suggesterItem, dark && styles.suggesterItemDark);
      item.dataset.value = key;
      const glyph = document.createElement('span');
      glyph.className = styles.emojiResult;
      glyph.textContent = emoji;
      item.append(glyph, ` ${key}`);
      list.append(item);
    }
    return list;
  }

  function options() {
    /** @type {HTMLElement[]} */
    const items = menu ? Array.from(menu.querySelectorAll('[role="option"]')) : [];
    return items.filter(visible);
  }

  function clearSelection() {
    textarea.removeAttribute('aria-activedescendant');
    menu?.querySelectorAll('[aria-selected="true"]').forEach((item) => item.removeAttribute('aria-selected'));
  }

  /** @param {1 | -1} step */
  function navigate(step) {
    const items = options();
    const current = items.find((item) => item.getAttribute('aria-selected') === 'true');
    const index = current ? items.indexOf(current) : -1;
    if ((index === items.length - 1 && step === 1) || (index === 0 && step === -1)) {
      clearSelection();
      textarea.focus();
      return;
    }
    let target = items[step === 1 ? 0 : items.length - 1];
    if (current && index >= 0 && index + step >= 0 && index + step < items.length) {
      target = items[index + step];
    }
    if (!target) {
      return;
    }
    for (const item of items) {
      if (item === target) {
        textarea.setAttribute('aria-activedescendant', target.id);
        target.setAttribute('aria-selected', 'true');
        target.scrollIntoView({ block: 'nearest', inline: 'nearest' });
      } else {
        item.removeAttribute('aria-selected');
      }
    }
  }

  /**
   * @param {HTMLUListElement} list
   * @param {number} at  where the name starts
   */
  function positionMenu(list, at) {
    const caret = caretRect(textarea, Math.min(at, textarea.value.length));
    const current = list.getBoundingClientRect();
    const left = caret.left - current.left;
    const top = caret.top + caret.height - current.top;
    if (left !== 0 || top !== 0) {
      const style = getComputedStyle(list);
      list.style.left = style.left ? `calc(${style.left} + ${left}px)` : `${left}px`;
      list.style.top = style.top ? `calc(${style.top} + ${top}px)` : `${top}px`;
    }
  }

  /** @returns {boolean} whether there was a list to close */
  function deactivate() {
    if (!menu) {
      return false;
    }
    const list = menu;
    menu = null;
    textarea.removeAttribute('aria-activedescendant');
    list.querySelectorAll('[aria-selected="true"]').forEach((item) => item.removeAttribute('aria-selected'));
    textarea.setAttribute('aria-expanded', 'false');
    list.removeEventListener('click', onListClick);
    list.removeEventListener('mousedown', onListMousedown);
    for (const name of ['role', 'aria-controls', 'aria-expanded', 'aria-autocomplete', 'aria-haspopup']) {
      textarea.removeAttribute(name);
    }
    list.remove();
    return true;
  }

  /**
   * @param {HTMLUListElement} list
   * @param {number} at  where the name starts
   */
  function activate(list, at) {
    if (textarea !== document.activeElement) {
      return;
    }
    deactivate();
    menu = list;
    composing = false;
    list.id = `text-expander-${Math.floor(Math.random() * 100000).toString()}`;
    host.append(list);
    textarea.setAttribute('role', 'combobox');
    textarea.setAttribute('aria-controls', list.id);
    textarea.setAttribute('aria-expanded', 'false');
    textarea.setAttribute('aria-autocomplete', 'list');
    textarea.setAttribute('aria-haspopup', 'listbox');
    positionMenu(list, at);
    textarea.setAttribute('aria-expanded', 'true');
    list.addEventListener('click', onListClick);
    list.addEventListener('mousedown', onListMousedown);
    navigate(1);
  }

  /** @param {Element} item */
  function commit(item) {
    if (!menu || !match || !(item instanceof HTMLElement)) {
      return;
    }
    const start = match.position - KEY.length;
    const end = match.position + match.text.length;
    const value = `:${item.dataset.value}: `;
    const cursor = start + value.length;
    deactivate();
    textarea.focus({ preventScroll: true });
    textarea.setSelectionRange(start, end);
    // as typing would, so the form keeps the text and undo takes the emoji back
    inserting = true;
    try {
      insertText(textarea, value, cursor, cursor);
    } finally {
      inserting = false;
    }
    lookBackIndex = cursor;
    match = null;
  }

  /** @param {MouseEvent} evt */
  function onListClick(evt) {
    const item = evt.target instanceof Element ? evt.target.closest('[role="option"]') : null;
    if (item && item.getAttribute('aria-disabled') !== 'true') {
      commit(item);
    }
  }

  function onListMousedown() {
    interactingWithList = true;
  }

  async function onInput() {
    // the selection goes on every input while the list is open, as the combobox cleared it
    clearSelection();
    if (inserting) {
      return;
    }
    if (justPasted) {
      justPasted = false;
      return;
    }
    const cursor = textarea.selectionEnd || 0;
    if (cursor <= lookBackIndex) {
      lookBackIndex = cursor - 1;
    }
    const found = findQuery(textarea.value, cursor, lookBackIndex);
    if (!found) {
      match = null;
      deactivate();
      return;
    }
    match = found;
    const list = await suggestions(found.text, props.theme);
    if (!match || found.position > textarea.value.length) {
      match = null;
      deactivate();
      return;
    }
    if (list) {
      activate(list, found.position);
    } else {
      deactivate();
    }
  }

  function onPaste() {
    justPasted = true;
  }

  function onBlur() {
    if (interactingWithList) {
      interactingWithList = false;
      return;
    }
    deactivate();
  }

  /** @param {CompositionEvent} evt */
  function onComposition(evt) {
    if (!menu) {
      return;
    }
    composing = evt.type === 'compositionstart';
    clearSelection();
  }

  /** @param {KeyboardEvent} evt */
  function onKeydown(evt) {
    if (evt.key === 'Escape') {
      match = null;
      if (deactivate()) {
        lookBackIndex = textarea.selectionEnd || lookBackIndex;
        evt.stopImmediatePropagation();
        evt.preventDefault();
      }
      return;
    }
    if (!menu) {
      return;
    }
    if (evt.shiftKey || evt.metaKey || evt.altKey || (!ctrlBindings && evt.ctrlKey) || composing) {
      return;
    }
    switch (evt.key) {
      case 'Enter':
      case 'Tab': {
        // a click on the option commits it, as the combobox clicked it
        const selected = menu.querySelector('[aria-selected="true"]');
        if (selected instanceof HTMLElement) {
          evt.preventDefault();
          if (selected.getAttribute('aria-disabled') !== 'true') {
            selected.click();
          }
        }
        break;
      }
      case 'ArrowDown':
        navigate(1);
        evt.preventDefault();
        break;
      case 'ArrowUp':
        navigate(-1);
        evt.preventDefault();
        break;
      case 'n':
      case 'p':
        if (ctrlBindings && evt.ctrlKey) {
          navigate(evt.key === 'n' ? 1 : -1);
          evt.preventDefault();
        }
        break;
      default:
        if (!evt.ctrlKey) {
          clearSelection();
        }
    }
  }

  textarea.addEventListener('paste', onPaste);
  textarea.addEventListener('input', onInput);
  textarea.addEventListener('keydown', onKeydown);
  textarea.addEventListener('blur', onBlur);
  textarea.addEventListener('compositionstart', onComposition);
  textarea.addEventListener('compositionend', onComposition);

  return {
    /**
     * @param {EmojiCompletionProps} next
     * @param {EmojiCompletionDeps} nextDeps
     */
    update(next, nextDeps) {
      props = next;
      deps = nextDeps;
    },
    destroy() {
      deactivate();
      match = null;
      textarea.removeEventListener('paste', onPaste);
      textarea.removeEventListener('input', onInput);
      textarea.removeEventListener('keydown', onKeydown);
      textarea.removeEventListener('blur', onBlur);
      textarea.removeEventListener('compositionstart', onComposition);
      textarea.removeEventListener('compositionend', onComposition);
    },
  };
}
