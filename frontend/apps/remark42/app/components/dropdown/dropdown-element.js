// A dropdown in plain JavaScript: a link-styled title button and, while open, a panel with the
// content. The content is built each time the panel opens and destroyed when it closes, so every
// opening starts from a fresh state, as the preact Dropdown's children did.
//
// A click outside closes the panel, and so does the embedding page's clickOutside message, unless
// the content keeps it open (a step that waits for something from outside the widget). Opening a
// top-level dropdown closes the others. While one is open, the page is kept tall enough to show it.

import { classes } from '../../common/dom.js';
import { isFromParent, parseMessage } from '../../utils/post-message.ts';
import { sleep } from '../../utils/sleep.ts';
import { buttonClassName, buttonElement } from '../button/button-element.js';
import styles from './dropdown.module.css';

/** @type {Set<{ close(): void }>} the top-level dropdowns that are open or opening */
const openTopLevel = new Set();

/**
 * @typedef {object} DropdownProps
 * @property {string} title
 * @property {string} [buttonTitle]
 * @property {'light' | 'dark'} theme
 * @property {boolean} [disabled]
 * @property {string} [mix]  an extra class for the root
 */

/**
 * @typedef {object} DropdownControl
 * @property {() => void} close  closes the panel
 * @property {(on: boolean) => void} keepOpen  while on, a click outside leaves the panel open
 */

/**
 * @typedef {object} DropdownContent
 * @property {() => void} destroy
 */

/**
 * @param {HTMLElement} root  an empty element; its attributes and children are this component's
 * @param {DropdownProps} initial
 * @param {(items: HTMLDivElement, control: DropdownControl) => DropdownContent} openContent
 *   builds the content into the element given, each time the panel opens
 */
export function dropdownElement(root, initial, openContent) {
  let props = initial;
  let active = false;
  let keptOpen = false;
  let translateX = 0;
  let destroyed = false;
  /** @type {DropdownContent | null} */
  let content = null;
  /** @type {ReturnType<typeof setInterval> | undefined} */
  let heightTimer;
  /** @type {string | null} */
  let storedHeight = null;
  let heightStored = false;

  const title = buttonElement({ kind: 'link', theme: props.theme, mix: styles.title });
  title.setAttribute('aria-haspopup', 'listbox');
  title.setAttribute('aria-expanded', 'false');
  const panel = document.createElement('div');
  panel.className = styles.content;
  panel.tabIndex = -1;
  panel.setAttribute('role', 'listbox');
  const items = document.createElement('div');
  items.className = styles.items;
  panel.append(items);
  root.append(title);

  const self = { close };
  /** @type {DropdownControl} */
  const control = {
    close,
    keepOpen(on) {
      keptOpen = on;
    },
  };

  title.addEventListener('click', toggle);
  // the verdict is reached in the capture phase, before the click reaches the content: a handler
  // there that redraws and detaches the clicked node would otherwise make it look outside
  document.addEventListener('click', onOutsideClick, { capture: true });
  window.addEventListener('message', onMessage);
  render();

  function render() {
    root.className = classes(
      styles.root,
      active && styles.active,
      props.theme === 'dark' ? styles.themeDark : styles.themeLight,
      props.mix
    );
    // the data attribute after the class, as the preact component wrote them
    root.dataset.dropdown = 'true';
    title.className = buttonClassName({ kind: 'link', theme: props.theme, mix: styles.title });
    // written out as false while closed, as preact writes an aria attribute given false
    title.setAttribute('aria-expanded', active ? 'true' : 'false');
    title.disabled = Boolean(props.disabled);
    if (props.buttonTitle === undefined) {
      title.removeAttribute('title');
    } else {
      title.title = props.buttonTitle;
    }
    if (title.textContent !== props.title) {
      title.textContent = props.title;
    }
    panel.style.transform = `translateX(${translateX}px)`;
  }

  function isNested() {
    let parent = root.parentElement;
    while (parent && parent !== document.body) {
      if (parent.hasAttribute('data-dropdown')) {
        return true;
      }
      parent = parent.parentElement;
    }
    return false;
  }

  async function toggle() {
    if (active) {
      close();
      return;
    }
    if (!isNested()) {
      openTopLevel.forEach((other) => other !== self && other.close());
      openTopLevel.add(self);
    }
    active = true;
    render();
    root.append(panel);
    content = openContent(items, control);

    await sleep(0);
    if (!active || destroyed) {
      return;
    }
    adjustPosition();
    watchHeight();
  }

  // moves the panel left when it would stick out past the right edge of the window, or right when
  // it starts left of the window
  function adjustPosition() {
    const rect = panel.getBoundingClientRect();
    if (rect.left > 0) {
      if (rect.right <= window.innerWidth) {
        return;
      }
      translateX = -Math.min(rect.left, rect.right - window.innerWidth);
    } else {
      translateX = -rect.left;
    }
    render();
  }

  // keeps the page tall enough for the open panel, which the iframe height follows
  function watchHeight() {
    if (isNested()) {
      return;
    }
    storedHeight = document.body.style.minHeight;
    heightStored = true;
    /** @type {number | null} */
    let previousBottom = null;
    heightTimer = setInterval(() => {
      if (!active) {
        return;
      }
      const rect = panel.getBoundingClientRect();
      const bottom = window.scrollY + Math.abs(rect.top) + panel.scrollHeight + 10;
      if (previousBottom === null && bottom <= window.innerHeight) {
        return;
      }
      if (bottom !== previousBottom) {
        previousBottom = bottom;
        document.body.style.minHeight = `${bottom}px`;
      }
    }, 100);
  }

  function close() {
    if (!active) {
      return;
    }
    active = false;
    keptOpen = false;
    translateX = 0;
    content?.destroy();
    content = null;
    items.replaceChildren();
    panel.remove();
    stopWatching();
    render();
  }

  function stopWatching() {
    openTopLevel.delete(self);
    clearInterval(heightTimer);
    heightTimer = undefined;
    if (heightStored) {
      document.body.style.minHeight = storedHeight ?? '';
      heightStored = false;
    }
  }

  /** @param {MouseEvent} evt */
  function onOutsideClick(evt) {
    if (!active || keptOpen || root.contains(/** @type {Node} */ (evt.target))) {
      return;
    }
    close();
  }

  /** @param {MessageEvent} evt */
  function onMessage(evt) {
    if (!isFromParent(evt) || !parseMessage(evt).clickOutside || !active || keptOpen) {
      return;
    }
    close();
  }

  return {
    /** @param {DropdownProps} next */
    update(next) {
      props = next;
      render();
    },
    destroy() {
      destroyed = true;
      // a panel kept open can be removed while open, for example when the reader signs out on the
      // token step, so it stops its timer and puts the page height back itself
      if (active) {
        active = false;
        content?.destroy();
        content = null;
      }
      stopWatching();
      document.removeEventListener('click', onOutsideClick, { capture: true });
      window.removeEventListener('message', onMessage);
    },
  };
}
