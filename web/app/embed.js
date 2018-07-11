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
      src="${BASE_URL}/web/iframe.html?${query}"
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

  const userInfo = {
    node: null,
    back: null,
    init(user) {
      if (!this.node) {
        this.node = document.createElement('div');
        this.node.style = `position: fixed; top: 0; right: 0; bottom: 0;width: 400px; transform: translate(400px, 0); transition: transform 0.4s ease-out; max-width: 100%`;
      }
      if (!this.back) {
        this.back = document.createElement('div');
        this.back.style = `position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.7);opacity: 0;transition: opacity 0.4s ease-out;`;
        this.back.onclick = () => this.close();
      }
      const queryUserInfo =
        query +
        '&page=user-info&' +
        `&id=${user.id}&name=${user.name}&picture=${user.picture || ''}&isDefaultPicture=${user.isDefaultPicture || 0}`;
      this.node.innerHTML = `
      <iframe
        src="${BASE_URL}/web/iframe.html?${queryUserInfo}"
        width="100%"
        height="100%"
        frameborder="0"
        allowtransparency="true"
        scrolling="no"
        tabindex="0"
        title="Remark42"
        verticalscrolling="no"
        horizontalscrolling="no"
      />
      `;
      document.body.appendChild(this.back);
      document.body.appendChild(this.node);
      setTimeout(() => {
        this.back.style.opacity = 1;
        this.node.style.transform = '';
      }, 400);
    },
    close() {
      if (this.node) {
        this.node.style.transform = 'translate(400px, 0)';
        this.node.remove();
      }
      if (this.back) {
        this.back.style.opacity = 0;
        this.back.remove();
      }
    },
  };

  function receiveMessages(event) {
    try {
      const data = typeof event.data === 'string' ? JSON.parse(event.data) : event.data;
      if (data.remarkIframeHeight) {
        iframe.style.height = `${data.remarkIframeHeight}px`;
      }

      if (data.scrollTo) {
        window.scrollTo(window.pageXOffset, data.scrollTo + iframe.getBoundingClientRect().top + window.pageYOffset);
      }

      if (data.hasOwnProperty('isUserInfoShown')) {
        if (data.isUserInfoShown) {
          userInfo.init(data.user || {});
        } else {
          userInfo.close();
        }
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
