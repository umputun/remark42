import { classes } from '../../../../common/dom.js';
import styles from './counter.module.css';

/**
 * The count of a profile's comments, drawn as the preact Counter drew it. The class is kept outside
 * the CSS modules so the browser suite can read the count: the production bundle hashes the module
 * names.
 *
 * @param {number} count
 */
export function counterElement(count) {
  const counter = document.createElement('div');
  counter.className = classes('comments-counter', styles.container);
  counter.dataset.testid = 'comments-counter';
  counter.textContent = String(count);
  return counter;
}
