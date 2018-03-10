import { BASE_URL, NODE_ID } from 'common/constants';

if (document.readyState !== 'interactive') {
  document.addEventListener('DOMContentLoaded', initEmbed);
} else {
  initEmbed();
}

function initEmbed() {
  const node = document.getElementById(NODE_ID);

  if (!node) {
    console.error('Remark42: Can\'t find root node.');
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

  remark_config.url = remark_config.url || window.location.href;

  const query = Object.keys(remark_config)
    .map(key => `${encodeURIComponent(key)}=${encodeURIComponent(remark_config[key])}`)
    .join('&');

  node.innerHTML = `
    <iframe
      src="${process.env.NODE_ENV === 'production' ? `${BASE_URL}/web` : ''}/iframe.html?${query}"
      width="100%"
      frameborder="0"
      allowtransparency="true"
      scrolling="no"
      tabindex="0"
      title="Remark42"
      style="width: 1px !important; min-width: 100% !important; border: none !important; overflow: hidden !important;"
      horizontalscrolling="no"
      verticalscrolling="no"
    ></iframe>
  `;

  const iframe = node.getElementsByTagName('iframe')[0];

  window.addEventListener('message', updateIframeHeight);

  function updateIframeHeight(event) {
    try {
      const data = JSON.parse(event.data);
      iframe.style.height = `${data.remarkIframeHeight}px`;
    } catch (e) {}
  }
}

