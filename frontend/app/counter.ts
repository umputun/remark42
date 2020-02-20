/* eslint-disable no-console */
declare let remark_config: CounterConfig;
import { COUNTER_NODE_CLASSNAME, BASE_URL, API_BASE } from '@app/common/constants.config';
import { CounterConfig } from '@app/common/config-types';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init(): void {
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
    const id = node.dataset.url || remark_config.url || window.location.origin + window.location.pathname;
    if (!acc[id]) acc[id] = [];
    acc[id].push(node);
    return acc;
  }, {});

  const oReq = new XMLHttpRequest();
  oReq.onreadystatechange = function(this: XMLHttpRequest) {
    if (this.readyState === XMLHttpRequest.DONE && this.status === 200) {
      try {
        const res = JSON.parse(this.responseText) as { url: string; count: number }[];
        res.forEach(item => map[item.url].map(n => (n.innerHTML = item.count.toString(10))));
      } catch (e) {}
    }
  };
  oReq.open('POST', `${BASE_URL}${API_BASE}/counts?site=${remark_config.site_id}`, true);
  oReq.setRequestHeader('Content-Type', 'application/json');
  oReq.send(JSON.stringify(Object.keys(map)));
}
