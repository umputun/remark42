import { classes } from '../../common/dom.js';
import { setAttributes } from '../../common/element.js';
import styles from './input.module.css';

/**
 * @typedef {object} InputLook
 * @property {string} [className]
 * @property {string} [type]  text when not given
 * @property {boolean} [invalid]
 */

/**
 * The class the preact Input gives a field.
 *
 * @param {InputLook} look
 */
export function inputClassName({ className, invalid }) {
  return classes(className, styles.input, invalid && styles.invalid);
}

/**
 * A field drawn as the preact Input draws it: the class and type first, then the attributes in
 * the order given. `value` is set as the property, so it is not written into the markup.
 *
 * @param {InputLook} look
 * @param {Record<string, import('../../common/element.js').AttributeValue>} [attributes]
 */
export function inputElement(look, attributes = {}) {
  const input = document.createElement('input');
  input.className = inputClassName(look);
  input.type = look.type ?? 'text';
  setAttributes(input, attributes);
  return input;
}
