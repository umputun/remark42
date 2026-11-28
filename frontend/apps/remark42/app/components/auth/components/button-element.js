import { classes } from '../../../common/dom.js';
import styles from './button.module.css';

/**
 * A button styled as the preact Button in this directory draws it. The caller sets its text,
 * handlers and disabled state.
 *
 * @param {{ kind?: 'transparent' | 'link' | 'hollow', size?: 'xs' | 'sm', className?: string }} look
 */
export function buttonElement({ kind, size, className }) {
  const button = document.createElement('button');
  button.className = classes(className, styles.button, kind && styles[kind], size && styles[size]);
  return button;
}
