// The last comments of a site, written into the host page: every element with the class
// remark42__last-comments gets the list. This entry is plain JavaScript on the DOM API and imports
// no package, which the build enforces.

import { API_BASE, BASE_URL } from './common/constants.config.ts';
import { requestURL } from './common/fetcher-core.ts';
import { renderList } from './components/list-comments/list-comments.js';
import { getLocale } from './utils/getLocale.ts';
import { loadLocale } from './utils/loadLocale.ts';

const LAST_COMMENTS_NODE_CLASSNAME = 'remark42__last-comments';
const DEFAULT_LAST_COMMENTS_MAX = 15;

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  const nodes = /** @type {HTMLCollectionOf<HTMLElement>} */ (
    document.getElementsByClassName(LAST_COMMENTS_NODE_CLASSNAME)
  );
  const config = window.remark_config;

  if (!config) {
    throw new Error('Remark42: Config object is undefined');
  }

  const { site_id, max_last_comments } = config;

  if (!site_id) {
    throw new Error('Remark42: Site ID is undefined.');
  }

  const styles = document.createElement('link');
  styles.href = `${BASE_URL}/web/last-comments.css`;
  styles.rel = 'stylesheet';
  (document.head || document.body).appendChild(styles);

  for (const node of Array.from(nodes)) {
    const max = (node.dataset.max && parseInt(node.dataset.max, 10)) || max_last_comments || DEFAULT_LAST_COMMENTS_MAX;
    const locale = getLocale(config);

    // the list shows no message of its own, so nothing in the catalogue is read. It is still
    // requested, and the list waits for it, as it did when the list was a component
    Promise.all([getLastComments(site_id, max), loadLocale(locale)]).then(([comments]) => {
      try {
        node.replaceChildren(renderList(comments, locale));
      } catch (e) {
        console.error('Remark42: Something went wrong with last comments rendering');
        console.error(e);
      }
    });
  }
}

/**
 * The feed is the same for every reader, so the request carries no session: it is a plain fetch,
 * which on a host page of another origin sends no cookie and no header of the widget's.
 *
 * @param {string} siteId
 * @param {number} max
 * @returns {Promise<import('./common/types.ts').Comment[]>}
 */
async function getLastComments(siteId, max) {
  const response = await fetch(requestURL(`${BASE_URL}${API_BASE}`, `/last/${max}`, {}, siteId));

  if (!response.ok) {
    throw new Error(`Remark42: the last comments request failed with status ${response.status}`);
  }

  return response.json();
}
