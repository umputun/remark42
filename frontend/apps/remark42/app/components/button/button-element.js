import { classes } from '../../common/dom.js';
import styles from './button.module.css';

const KIND = { primary: styles.kindPrimary, secondary: styles.kindSecondary, link: styles.kindLink };
const SIZE = { middle: styles.sizeMiddle, large: styles.sizeLarge };

/**
 * @typedef {object} ButtonLook
 * @property {'primary' | 'secondary' | 'link'} [kind]
 * @property {'middle' | 'large'} [size]
 * @property {'light' | 'dark'} [theme]
 * @property {string | string[]} [mix]
 * @property {string} [className]
 * @property {'button' | 'submit' | 'reset'} [type]  button when not given
 */

/**
 * The class the preact Button in this directory gives a button.
 *
 * @param {ButtonLook} look
 */
export function buttonClassName({ kind, size, theme, mix, className }) {
  return classes(
    styles.root,
    kind && KIND[kind],
    size && SIZE[size],
    theme === 'dark' && styles.themeDark,
    ...(Array.isArray(mix) ? mix : [mix]),
    className
  );
}

/**
 * A button drawn as the preact Button in this directory draws it: the class, then the type. The
 * caller sets the text, handlers and any further attributes.
 *
 * @param {ButtonLook} look
 */
export function buttonElement(look) {
  const button = document.createElement('button');
  button.className = buttonClassName(look);
  button.type = look.type ?? 'button';
  return button;
}
