/* eslint-disable no-console, @typescript-eslint/camelcase */
/** @jsx createElement */
declare let remark_config: LastCommentsConfig;
// Must be the first import
if (process.env.NODE_ENV === 'development') {
  // Must use require here as import statements are only allowed
  // to exist at the top of a file.
  require('preact/debug');
}
import loadPolyfills from '@app/common/polyfills';
import { createElement, render } from 'preact';
import { IntlProvider } from 'react-intl';

import getLastComments from '@app/common/api.getLastComments';
import { LastCommentsConfig } from '@app/common/config-types';
import { BASE_URL } from '@app/common/constants.config';
import { loadLocale } from '@app/utils/loadLocale';
import { getLocale } from '@app/utils/getLocale';
import { ListComments } from '@app/components/list-comments';

const LAST_COMMENTS_NODE_CLASSNAME = 'remark42__last-comments';
const DEFAULT_LAST_COMMENTS_MAX = 15;

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

async function init(): Promise<void> {
  __webpack_public_path__ = BASE_URL + '/web/';

  await loadPolyfills();

  const nodes = document.getElementsByClassName(LAST_COMMENTS_NODE_CLASSNAME);

  if (!nodes) {
    console.error("Remark42: Can't find last comments nodes.");
    return;
  }

  try {
    remark_config = remark_config || {};
  } catch (e) {
    console.error('Remark42: Config object is undefined.');
    return;
  }

  if (!remark_config.site_id) {
    console.error('Remark42: Site ID is undefined.');
    return;
  }

  const styles = document.createElement('link');
  styles.href = `${BASE_URL}/web/last-comments.css`;
  styles.rel = 'stylesheet';
  (document.head || document.body).appendChild(styles);

  ([].slice.call(nodes) as HTMLElement[]).forEach(node => {
    const max =
      (node.dataset.max && parseInt(node.dataset.max, 10)) ||
      remark_config.max_last_comments ||
      DEFAULT_LAST_COMMENTS_MAX;
    const locale = getLocale(remark_config);
    Promise.all([getLastComments(remark_config.site_id!, max), loadLocale(locale)]).then(([comments, messages]) => {
      try {
        render(
          <IntlProvider locale={locale} messages={messages}>
            <ListComments comments={comments} />
          </IntlProvider>,
          node
        );
      } catch (e) {
        console.error('Remark42: Something went wrong with last comments rendering');
        console.error(e);
      }
    });
  });
}
