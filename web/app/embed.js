/* eslint-disable no-console */
import { BASE_URL, NODE_ID, COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  const node = document.getElementById(NODE_ID);

  if (!node) {
    console.error("Remark42: Can't find root node.");
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

  remark_config.url = (remark_config.url || window.location.href).split('#')[0];

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

  window.addEventListener('message', receiveMessages);

  window.addEventListener('hashchange', postHashToIframe);

  setTimeout(postHashToIframe, 1000);

  function receiveMessages(event) {
    try {
      const data = typeof event.data === 'string' ? JSON.parse(event.data) : event.data;
      if (data.remarkIframeHeight) {
        iframe.style.height = `${data.remarkIframeHeight}px`;
      }

      if (data.scrollTo) {
        window.scrollTo(window.pageXOffset, data.scrollTo + iframe.getBoundingClientRect().top + window.pageYOffset);
      }
    } catch (e) {}
  }

  function postHashToIframe(e) {
    const hash = e ? `#${e.newURL.split('#')[1]}` : window.location.hash;

    if (hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
      if (e) e.preventDefault();

      iframe.contentWindow.postMessage(JSON.stringify({ hash }), '*');
    }
  }
}
