// The RSS subscription in plain JavaScript: a dropdown with the feeds of this page, of the whole
// site and of the replies to the reader.

import { API_BASE, BASE_URL } from '../../../common/constants.config.ts';
import { defineMessages } from '../../../common/intl-message.ts';
import { siteId, url } from '../../../common/settings.ts';
import { dropdownElement } from '../../dropdown/dropdown-element.js';
import { dropdownItemElement } from '../../dropdown/dropdown-item-element.js';
import styles from './subscribe-by-rss.module.css';

/**
 * The address of a feed.
 *
 * @param {'post' | 'site' | 'reply'} type
 * @param {string} [urlParams]
 */
export function createSubscribeUrl(type, urlParams = '') {
  return `${BASE_URL}${API_BASE}/rss/${type}?site=${siteId}${urlParams}`;
}

/**
 * @typedef {object} RssProps
 * @property {string | null} userId
 * @property {'light' | 'dark'} theme
 */

/**
 * @param {HTMLElement} root  an empty div; its attributes and children are this component's
 * @param {RssProps} initial
 * @param {import('../../../common/intl-message.ts').IntlShape} initialIntl
 */
export function mountSubscribeByRss(root, initial, initialIntl) {
  let props = initial;
  let intl = initialIntl;

  const dropdown = dropdownElement(root, dropdownProps(), (items) => {
    items.replaceChildren(
      ...feeds().map(([href, label]) => {
        const link = document.createElement('a');
        link.href = href;
        link.className = styles.link;
        link.target = '_blank';
        link.rel = 'noreferrer';
        link.textContent = label;
        return dropdownItemElement([link]);
      })
    );
    return { destroy() {} };
  });

  function dropdownProps() {
    return {
      title: intl.formatMessage(messages.title),
      buttonTitle: intl.formatMessage(messages.buttonTitle),
      mix: styles.rssDropdown,
      theme: props.theme,
    };
  }

  /** @returns {Array<[string, string]>} */
  function feeds() {
    return [
      [createSubscribeUrl('post', `&url=${url}`), intl.formatMessage(messages.thread)],
      [createSubscribeUrl('site'), intl.formatMessage(messages.site)],
      [createSubscribeUrl('reply', `&user=${props.userId}`), intl.formatMessage(messages.replies)],
    ];
  }

  return {
    /**
     * @param {RssProps} next
     * @param {import('../../../common/intl-message.ts').IntlShape} [nextIntl]
     */
    update(next, nextIntl = intl) {
      props = next;
      intl = nextIntl;
      dropdown.update(dropdownProps());
    },
    destroy() {
      dropdown.destroy();
    },
  };
}

const messages = defineMessages({
  thread: {
    id: 'subscribeByRSS.thread',
    defaultMessage: 'This page',
  },
  site: {
    id: 'subscribeByRSS.site',
    defaultMessage: 'All pages',
  },
  replies: {
    id: 'subscribeByRSS.replies',
    defaultMessage: 'Replies',
  },
  buttonTitle: {
    id: 'subscribeByRSS.button-title',
    defaultMessage: 'Subscribe by RSS',
  },
  title: {
    id: 'subscribeByRSS.title',
    defaultMessage: 'RSS',
  },
});
