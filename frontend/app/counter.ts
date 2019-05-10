/* eslint-disable no-console */
declare let remark_config: CounterConfig;

import loadPolyfills from '@app/common/polyfills';
import api from './common/api';
import { COUNTER_NODE_CLASSNAME } from '@app/common/constants';
import { CounterConfig } from '@app/common/config-types';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

async function init(): Promise<void> {
  __webpack_public_path__ = remark_config.host + '/web/';

  await loadPolyfills();

  const nodes: HTMLElement[] = [].slice.call(document.getElementsByClassName(COUNTER_NODE_CLASSNAME));

  if (!nodes) {
    console.error("Remark42: Can't find counter nodes.");
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

  const map = nodes.reduce<{ [key: string]: HTMLElement[] }>((acc, node) => {
    const id = node.dataset.url || remark_config.url || window.location.href;
    if (!acc[id]) acc[id] = [];
    acc[id].push(node);
    return acc;
  }, {});

  api.getCommentsCount(remark_config.site_id, Object.keys(map)).then(res => {
    res.forEach(item => map[item.url].map(n => (n.innerHTML = item.count.toString(10))));
  });
}
