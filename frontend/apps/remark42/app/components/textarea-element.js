import { setAttributes } from '../common/element.js';

/** @param {HTMLTextAreaElement} textarea */
function autoResize(textarea) {
  textarea.style.height = '';
  textarea.style.height = `${textarea.scrollHeight}px`;
}

/**
 * @typedef {object} TextareaOptions
 * @property {(evt: Event) => void} [onInput]
 * @property {boolean} [controlled]  the caller passes the text back through setValue after each
 *   input, and the height follows what it passes
 */

/**
 * A textarea that grows with its text, as the preact TextareaAutosize does: the attributes in the
 * order given, then `dir="auto"`. resize() measures it again; call it once the textarea is on the
 * page, since a detached one has no height to measure.
 *
 * @param {Record<string, import('../common/element.js').AttributeValue>} attributes
 * @param {TextareaOptions} [options]
 */
export function textareaElement(attributes, { onInput, controlled = false } = {}) {
  const textarea = document.createElement('textarea');
  setAttributes(textarea, attributes);
  if (attributes.id !== undefined) {
    textarea.dataset.testid = String(attributes.id);
  }
  textarea.setAttribute('dir', 'auto');
  textarea.addEventListener('input', (evt) => {
    if (!onInput || !controlled) {
      autoResize(textarea);
    }
    onInput?.(evt);
  });

  return {
    textarea,
    resize() {
      autoResize(textarea);
    },
    /**
     * The text, set when it differs from what the field holds, and the height measured again.
     *
     * @param {string} value
     */
    setValue(value) {
      if (textarea.value !== value) {
        textarea.value = value;
      }
      autoResize(textarea);
    },
  };
}
