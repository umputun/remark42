// The Telegram sign-in step in plain JavaScript: how to start the bot, the QR code on a wide
// screen, the error and the Check button. The nodes sit side by side in the caller's element,
// as the preact TelegramLink's fragment did; nodes() lists them in order after each update.

import { classes } from '../../common/dom.js';
import { API_BASE, BASE_URL } from '../../common/constants.config.ts';
import { buttonElement } from '../button/button-element.js';
import { messages } from './telegram.messages.ts';
import styles from './telegram-link.module.css';

// the paragraph is assembled from separate messages, and Japanese and Chinese put no spaces
// between words, so joining their fragments with one would put spaces mid-sentence
const SPACELESS_LOCALES = new Set(['ja', 'zh', 'zh-tw']);

/**
 * @typedef {object} TelegramLinkProps
 * @property {string} bot
 * @property {string} token
 * @property {(evt: Event) => void} onSubmit  the Check button was pressed
 * @property {string | null} [errorMessage]
 */

/**
 * @param {TelegramLinkProps} initial
 * @param {import('../../common/intl-message.ts').IntlShape} initialIntl
 */
export function telegramLink(initial, initialIntl) {
  let props = initial;
  let intl = initialIntl;

  const paragraph = document.createElement('p');
  paragraph.className = classes('telegram', styles.telegram);
  const first = document.createTextNode('');
  const gapBefore = document.createTextNode('');
  const link = document.createElement('a');
  link.target = '_blank';
  link.rel = 'noopener noreferrer';
  link.href = '';
  const qrNote = document.createTextNode('');
  const gapAfter = document.createTextNode('');
  const second = document.createTextNode('');
  const third = document.createTextNode('');
  paragraph.append(first, gapBefore, link, qrNote, gapAfter, second, document.createElement('br'), third);

  const qr = document.createElement('img');
  qr.src = '';
  qr.height = 200;
  qr.width = 200;
  qr.className = classes('telegram-qr', styles.telegramQR);
  qr.alt = '';

  const error = document.createElement('div');
  error.className = classes('auth-error', styles.error);

  const check = buttonElement({
    kind: 'primary',
    size: 'large',
    className: classes('auth-submit', styles.button),
  });
  check.addEventListener('click', (evt) => props.onSubmit(evt));

  /** @type {Node[]} */
  let nodes = [];
  render();

  function render() {
    const address = `https://t.me/${props.bot}/?start=${props.token}`;
    const gap = SPACELESS_LOCALES.has(intl.locale) ? '' : ' ';
    const wide = window.screen.width >= 768;

    first.data = intl.formatMessage(messages.telegramMessage1);
    gapBefore.data = gap;
    link.href = address;
    link.textContent = intl.formatMessage(messages.telegramLink);
    qrNote.data = wide ? `${gap}${intl.formatMessage(messages.telegramOptionalQR)}` : '';
    gapAfter.data = gap;
    second.data = intl.formatMessage(messages.telegramMessage2);
    third.data = intl.formatMessage(messages.telegramMessage3);
    // only on a screen that shows it: an image loads its source even while it is off the page, and
    // the server makes a PNG for each request
    const source = `${BASE_URL}${API_BASE}/qr/telegram?url=${address}`;
    if (wide && qr.getAttribute('src') !== source) {
      qr.src = source;
    }
    qr.alt = intl.formatMessage(messages.telegramQR);
    error.textContent = props.errorMessage ?? '';
    check.textContent = intl.formatMessage(messages.telegramCheck);

    nodes = [paragraph];
    if (wide) {
      nodes.push(qr);
    }
    if (props.errorMessage) {
      nodes.push(error);
    }
    nodes.push(check);
  }

  return {
    /** the nodes to place, in order */
    nodes: () => nodes,
    /**
     * @param {TelegramLinkProps} next
     * @param {import('../../common/intl-message.ts').IntlShape} [nextIntl]
     */
    update(next, nextIntl = intl) {
      props = next;
      intl = nextIntl;
      render();
    },
  };
}
