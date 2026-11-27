import { classes } from '../../../common/dom.js';
import styles from './button.module.css';

/**
 * @typedef {object} ButtonLook
 * @property {'transparent' | 'link' | 'hollow'} [kind]
 * @property {'xs' | 'sm'} [size]
 * @property {string} [className]
 * @property {boolean} [selected]
 */

/**
 * The class the preact Button in this directory gives a button.
 *
 * @param {ButtonLook} look
 */
export function buttonClassName({ kind, size, className, selected }) {
  return classes(className, styles.button, kind && styles[kind], size && styles[size], selected && styles.selected);
}

/**
 * A button styled as the preact Button in this directory draws it. The caller sets its text,
 * handlers and disabled state, and appends a suffix in a `suffix` element of its own
 * (`suffixElement`).
 *
 * @param {ButtonLook} look
 */
export function buttonElement(look) {
  const button = document.createElement('button');
  button.className = buttonClassName(look);
  return button;
}

/**
 * The box the preact Button puts its suffix in, after the button's own content.
 *
 * @param {Node} content
 */
export function suffixElement(content) {
  const suffix = document.createElement('div');
  suffix.className = styles.suffix;
  suffix.append(content);
  return suffix;
}
