import { h, render } from 'preact';

import { BASE_URL, DEFAULT_LAST_COMMENTS_MAX, LAST_COMMENTS_NODE_CLASSNAME } from './common/constants';

import api from 'common/api';

import ListComments from 'components/list-comments';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  const nodes = document.getElementsByClassName(LAST_COMMENTS_NODE_CLASSNAME);

  if (!nodes) {
    console.error('Remark42: Can\'t find last comments nodes.');
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
  styles.href = `${BASE_URL}/web/remark.css`;
  styles.rel = 'stylesheet';
  (document.head || document.body).appendChild(styles);

  [].slice.call(nodes).forEach(node => {
    const max = node.dataset.max || remark_config.max_last_comments || DEFAULT_LAST_COMMENTS_MAX;
    api.getLastComments({ max, siteId: remark_config.site_id })
      .then(comments => {
        try {
          render(<ListComments comments={comments}/>, node);
        } catch (e) {
          console.error('Remark42: Something went wrong with last comments rendering');
          console.error(e);
        }
      });
  });
}

