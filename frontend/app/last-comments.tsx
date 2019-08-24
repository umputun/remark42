/* eslint-disable no-console, @typescript-eslint/camelcase */
/** @jsx h */
declare let remark_config: LastCommentsConfig;

import loadPolyfills from '@app/common/polyfills';
import '@app/utils/patchPreactContext';
import { h, render } from 'preact';
import 'preact/debug';
import { getLastComments } from './common/api';
import { LastCommentsConfig } from '@app/common/config-types';
import { BASE_URL, DEFAULT_LAST_COMMENTS_MAX, LAST_COMMENTS_NODE_CLASSNAME } from '@app/common/constants';
import { ListComments } from '@app/components/list-comments';

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
    getLastComments(remark_config.site_id!, max).then(comments => {
      try {
        render(<ListComments comments={comments} />, node);
      } catch (e) {
        console.error('Remark42: Something went wrong with last comments rendering');
        console.error(e);
      }
    });
  });
}
