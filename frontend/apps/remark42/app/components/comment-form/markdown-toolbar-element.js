// The markdown toolbar above the comment field, in plain JavaScript. It draws the markup the
// preact toolbar drew with the toolbar package's elements, and does what that package did for the
// buttons the form has: a button applies its markdown to the field's selection on a click, Space
// or Enter, and the arrow keys, Home and End move between the buttons. The elements are not
// registered as custom elements; their tag names stay for the pages that style them.

import { defineMessages } from '../../common/intl-message.ts';
import { insertText } from './insert-text.js';
import { formatSelection, STYLES } from './markdown-format.js';
import { toolbarIcons } from './markdown-toolbar-icons.js';
import styles from './markdown-toolbar.module.css';

export const messages = defineMessages({
  bold: { id: 'toolbar.bold', defaultMessage: 'Bold' },
  header: { id: 'toolbar.header', defaultMessage: 'Heading' },
  italic: { id: 'toolbar.italic', defaultMessage: 'Italic' },
  quote: { id: 'toolbar.quote', defaultMessage: 'Quote' },
  code: { id: 'toolbar.code', defaultMessage: 'Code' },
  link: { id: 'toolbar.link', defaultMessage: 'Link' },
  unorderedList: { id: 'toolbar.unordered-list', defaultMessage: 'Bulleted list' },
  orderedList: { id: 'toolbar.ordered-list', defaultMessage: 'Numbered list' },
  attachImage: {
    id: 'toolbar.attach-image',
    defaultMessage: 'Attach an image, drag and drop one, or paste one from the clipboard',
  },
});

/**
 * The buttons in their groups, each with its element name, style, label and icon.
 *
 * @type {Array<Array<{ tag: string, style: string, label: keyof typeof messages, icon: keyof typeof toolbarIcons }>>}
 */
const GROUPS = [
  [
    { tag: 'md-header', style: 'header', label: 'header', icon: 'header' },
    { tag: 'md-bold', style: 'bold', label: 'bold', icon: 'bold' },
    { tag: 'md-italic', style: 'italic', label: 'italic', icon: 'italic' },
  ],
  [
    { tag: 'md-quote', style: 'quote', label: 'quote', icon: 'quote' },
    { tag: 'md-code', style: 'code', label: 'code', icon: 'code' },
    { tag: 'md-link', style: 'link', label: 'link', icon: 'link' },
  ],
  [
    { tag: 'md-unordered-list', style: 'unordered-list', label: 'unorderedList', icon: 'unorderedList' },
    { tag: 'md-ordered-list', style: 'ordered-list', label: 'orderedList', icon: 'orderedList' },
  ],
];

/**
 * @typedef {object} ToolbarProps
 * @property {string} textareaId  the field the toolbar edits, named in its `for` attribute
 * @property {boolean} allowUpload  whether the toolbar offers to attach an image
 */

/**
 * @typedef {object} ToolbarDeps
 * @property {import('../../common/intl-message.ts').IntlShape} intl
 * @property {() => HTMLTextAreaElement | null} field  the field the buttons edit
 * @property {(files: File[]) => Promise<void>} uploadImages
 */

/**
 * The buttons a key can move between: those on the page, in order. One that takes no space is
 * skipped, as the package skipped it.
 *
 * @param {HTMLElement} toolbar
 * @returns {HTMLElement[]}
 */
function visibleButtons(toolbar) {
  /** @type {HTMLElement[]} */
  const buttons = Array.from(
    toolbar.querySelectorAll(
      GROUPS.flat()
        .map((button) => button.tag)
        .join(', ')
    )
  );
  return buttons.filter((button) => !button.hidden && (button.offsetWidth > 0 || button.offsetHeight > 0));
}

/**
 * @param {ToolbarProps} initial
 * @param {ToolbarDeps} initialDeps
 */
export function markdownToolbar(initial, initialDeps) {
  let props = initial;
  let deps = initialDeps;

  const root = document.createElement('markdown-toolbar');
  root.className = styles.toolbar;
  root.setAttribute('for', props.textareaId);
  root.setAttribute('role', 'toolbar');
  root.setAttribute('tabindex', '0');

  /** @type {Array<{ node: HTMLElement, label: keyof typeof messages }>} */
  const labelled = [];

  /**
   * @param {string} style
   * @param {Event} [evt]
   */
  function apply(style, evt) {
    evt?.preventDefault();
    const field = deps.field();
    if (!field) {
      return;
    }
    field.focus();
    const edit = formatSelection(field.value, field.selectionStart, field.selectionEnd, STYLES[style]);
    field.setSelectionRange(edit.start, edit.end);
    insertText(field, edit.text, edit.selectionStart, edit.selectionEnd);
  }

  const groups = GROUPS.map((buttons) => {
    const group = document.createElement('div');
    group.className = styles.group;
    for (const { tag, style, label, icon } of buttons) {
      const button = document.createElement(tag);
      button.className = styles.item;
      button.addEventListener('click', (evt) => apply(style, evt));
      button.addEventListener('keydown', (evt) => {
        if (evt.key === ' ' || evt.key === 'Enter') {
          apply(style, evt);
        }
      });
      button.append(toolbarIcons[icon]());
      labelled.push({ node: button, label });
      group.append(button);
    }
    root.append(group);
    return group;
  });

  const upload = document.createElement('label');
  upload.className = styles.item;
  labelled.push({ node: upload, label: 'attachImage' });
  const fileInput = document.createElement('input');
  fileInput.multiple = true;
  fileInput.className = styles.fileInput;
  fileInput.type = 'file';
  fileInput.addEventListener('change', async () => {
    const files = fileInput.files;
    if (!(props.allowUpload && files && files.length !== 0)) {
      return;
    }
    await deps.uploadImages(Array.from(files));
    fileInput.value = '';
  });
  upload.append(fileInput, toolbarIcons.image());

  /** @param {KeyboardEvent} evt */
  function onKeydown(evt) {
    const { key } = evt;
    if (key !== 'ArrowRight' && key !== 'ArrowLeft' && key !== 'Home' && key !== 'End') {
      return;
    }
    const buttons = visibleButtons(root);
    const index = buttons.indexOf(/** @type {HTMLElement} */ (evt.target));
    const { length } = buttons;
    if (index === -1) {
      return;
    }
    let next = 0;
    if (key === 'ArrowLeft') next = index - 1;
    if (key === 'ArrowRight') next = index + 1;
    if (key === 'End') next = length - 1;
    if (next < 0) next = length - 1;
    if (next > length - 1) next = 0;
    buttons.forEach((button, i) => button.setAttribute('tabindex', i === next ? '0' : '-1'));
    evt.preventDefault();
    buttons[next].focus();
  }

  // the first time the toolbar itself takes the focus, it hands it to its first button and the
  // buttons take turns in the tab order from then on
  function onFirstFocus() {
    root.removeAttribute('tabindex');
    let tabindex = '0';
    for (const button of visibleButtons(root)) {
      button.setAttribute('tabindex', tabindex);
      if (tabindex === '0') {
        button.focus();
        tabindex = '-1';
      }
    }
  }

  root.addEventListener('keydown', onKeydown);
  root.addEventListener('focus', onFirstFocus, { once: true });

  function render() {
    for (const { node, label } of labelled) {
      const text = deps.intl.formatMessage(messages[label]);
      node.title = text;
      node.setAttribute('aria-label', text);
    }
    if (props.allowUpload && upload.parentNode !== groups[1]) {
      groups[1].append(upload);
    } else if (!props.allowUpload) {
      upload.remove();
    }
  }

  render();

  return {
    root,
    /**
     * @param {ToolbarProps} next
     * @param {ToolbarDeps} nextDeps
     */
    update(next, nextDeps) {
      props = next;
      deps = nextDeps;
      root.setAttribute('for', props.textareaId);
      render();
    },
    destroy() {
      root.removeEventListener('keydown', onKeydown);
      root.removeEventListener('focus', onFirstFocus);
      root.removeAttribute('tabindex');
    },
  };
}
