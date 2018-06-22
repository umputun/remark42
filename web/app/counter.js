import { COUNTER_NODE_CLASSNAME } from './common/constants';

import api from 'common/api';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  const nodes = [].slice.call(document.getElementsByClassName(COUNTER_NODE_CLASSNAME));

  if (!nodes) {
    console.error('Remark42: Can\'t find counter nodes.');
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

  const map = nodes.reduce((acc, node) => {
    acc[node.dataset.url || remark_config.url || window.location.href] = node;
    return acc;
  }, {});

  api.counts({ urls: Object.keys(map), siteId: remark_config.site_id })
    .then(res => {
      res.forEach(item => (map[item.url].innerHTML = item.count));
    });
}

