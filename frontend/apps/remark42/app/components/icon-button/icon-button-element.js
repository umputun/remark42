import { classes } from '../../common/dom.js';
import styles from './icon-button.module.css';

/**
 * A button holding an icon, drawn as the preact IconButton draws it. The caller sets the title,
 * the icon and the handler.
 *
 * @param {string} [className]
 */
export function iconButtonElement(className) {
  const button = document.createElement('button');
  button.className = classes('icon-button', className, styles.root);
  return button;
}
