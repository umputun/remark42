import { COUNTER_NODE_CLASSNAME } from './common/constants'

import api from 'common/api';

if (document.readyState !== 'interactive') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  const nodes = document.getElementsByClassName(COUNTER_NODE_CLASSNAME);

  if (!nodes) {
    console.error('Remark42: Can\'t find counter nodes.');
    return;
  }

  try {
    remark_config = remark_config || {}
  } catch (e) {
    console.error('Remark42: Config object is undefined.');
    return;
  }

  if (!remark_config.site_id) {
    console.error('Remark42: Site ID is undefined.');
    return;
  }

  [].slice.call(nodes).forEach(node => {
    const url = node.dataset.url || remark_config.url || window.location.href;
    api.count({ url, siteId: remark_config.site_id })
      .then(({ count }) => { node.innerHTML = count; });
  });
}

