import { classes } from '../../common/dom.js';
import { messages } from '../auth/auth.messages.ts';
import styles from './spinner.module.css';

/**
 * The loading spinner, drawn as the preact Spinner draws it.
 *
 * @param {import('../../common/intl-message.ts').IntlShape} intl
 * @param {'white' | 'gray'} [color]
 */
export function spinnerElement(intl, color) {
  const spinner = document.createElement('div');
  spinner.className = classes('spinner', styles.root, color === 'gray' && styles.dark);
  spinner.setAttribute('role', 'presentation');
  spinner.setAttribute('aria-label', intl.formatMessage(messages.loading));
  spinner.dataset.testid = 'spinner';
  return spinner;
}
