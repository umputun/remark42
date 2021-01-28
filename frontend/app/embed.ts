import type { UserInfo, Theme } from 'common/types';
import { BASE_URL, NODE_ID, COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants.config';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function removeDomNode(node: HTMLElement | null) {
  if (node && node.parentNode) {
    node.parentNode.removeChild(node);
  }
}

function createFrame({
  host,
  query,
  height,
  __colors__ = {},
}: {
  host: string;
  query: string;
  height?: string;
  __colors__?: Record<string, string>;
}) {
  const iframe = document.createElement('iframe');

  iframe.src = `${host}/web/iframe.html?${query}`;
  iframe.name = JSON.stringify({ __colors__ });
  iframe.setAttribute('width', '100%');
  iframe.setAttribute('frameborder', '0');
  iframe.setAttribute('allowtransparency', 'true');
  iframe.setAttribute('scrolling', 'no');
  iframe.setAttribute('tabindex', '0');
  iframe.setAttribute('title', 'Comments | Remark42');
  iframe.setAttribute('horizontalscrolling', 'no');
  iframe.setAttribute('verticalscrolling', 'no');
  iframe.setAttribute(
    'style',
    'width: 1px !important; min-width: 100% !important; border: none !important; overflow: hidden !important;'
  );

  if (height) {
    iframe.setAttribute('height', height);
  }

  return iframe;
}

function init() {
  window.REMARK42 = window.REMARK42 || {};
  window.REMARK42.createInstance = createInstance;

  createInstance(window.remark_config);

  window.dispatchEvent(new Event('REMARK42::ready'));
}

function createInstance(config: typeof window.remark_config) {
  const root = document.getElementById(NODE_ID);

  if (!root) {
    throw new Error("Remark42: Can't find root node.");
  }
  if (!window.remark_config) {
    throw new Error('Remark42: Config object is undefined.');
  }
  if (!window.remark_config.site_id) {
    throw new Error('Remark42: Site ID is undefined.');
  }

  let initDataAnimationTimeout: number | null = null;
  let titleObserver: MutationObserver | null = null;

  config.url = (config.url || `${window.location.origin}${window.location.pathname}`).split('#')[0];

  const query = Object.keys(config)
    .filter((key) => key !== '__colors__')
    .map(
      (key) =>
        `${encodeURIComponent(key)}=${encodeURIComponent(
          config[key as keyof Omit<typeof window.remark_config, '__colors__'>] as string | number | boolean
        )}`
    )
    .join('&');

  const iframe =
    (root.firstElementChild as HTMLIFrameElement) ||
    createFrame({ host: BASE_URL, query, __colors__: config.__colors__ });

  root.appendChild(iframe);

  window.addEventListener('message', receiveMessages);
  window.addEventListener('hashchange', postHashToIframe);
  document.addEventListener('click', postClickOutsideToIframe);

  const titleElement = document.querySelector('title');
  if (titleElement) {
    titleObserver = new MutationObserver((mutations) => postTitleToIframe(mutations[0].target.textContent!));
    titleObserver.observe(titleElement, {
      subtree: true,
      characterData: true,
      childList: true,
    });
  }

  const remarkRootId = 'remark-km423lmfdslkm34';
  const userInfo: {
    node: HTMLElement | null;
    back: HTMLElement | null;
    closeEl: HTMLElement | null;
    iframe: HTMLIFrameElement | null;
    style: HTMLStyleElement | null;
    init: (user: UserInfo) => void;
    close: () => void;
    delay: number | null;
    events: string[];
    onAnimationClose: () => void;
    onKeyDown: (e: KeyboardEvent) => void;
    animationStop: () => void;
    remove: () => void;
  } = {
    node: null,
    back: null,
    closeEl: null,
    iframe: null,
    style: null,
    init(user) {
      this.animationStop();
      if (!this.style) {
        this.style = document.createElement('style');
        this.style.setAttribute('rel', 'stylesheet');
        this.style.setAttribute('type', 'text/css');
        this.style.innerHTML = `
        #${remarkRootId}-node {
          position: fixed;
          top: 0;
          right: 0;
          bottom: 0;
          width: 400px;
          transition: transform 0.4s ease-out;
          max-width: 100%;
          transform: translate(400px, 0);
        }
        #${remarkRootId}-node[data-animation] {
          transform: translate(0, 0);
        }
        #${remarkRootId}-back {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: rgba(0,0,0,0.7);
          opacity: 0;
          transition: opacity 0.4s ease-out;
        }
        #${remarkRootId}-back[data-animation] {
          opacity: 1;
        }
        #${remarkRootId}-close {
          top: 0px;
          right: 400px;
          position: absolute;
          text-align: center;
          font-size: 25px;
          cursor: pointer;
          color: white;
          border-color: transparent;
          border-width: 0;
          padding: 0;
          margin-right: 4px;
          background-color: transparent;
        }
        @media all and (max-width: 430px) {
          #${remarkRootId}-close {
            right: 0px;
            font-size: 20px;
            color: black;
          }
        }
      `;
      }
      if (!this.node) {
        this.node = document.createElement('div');
        this.node.id = `${remarkRootId}-node`;
      }
      if (!this.back) {
        this.back = document.createElement('div');
        this.back.id = `${remarkRootId}-back`;
        this.back.onclick = () => this.close();
      }
      if (!this.closeEl) {
        this.closeEl = document.createElement('button');
        this.closeEl.id = `${remarkRootId}-close`;
        this.closeEl.innerHTML = '&#10006;';
        this.closeEl.onclick = () => this.close();
      }
      const queryUserInfo = `${query}&page=user-info&&id=${user.id}&name=${user.name}&picture=${
        user.picture || ''
      }&isDefaultPicture=${user.isDefaultPicture || 0}`;
      const iframe = createFrame({ host: BASE_URL, query: queryUserInfo, height: '100%' });
      this.node.appendChild(iframe);
      this.iframe = iframe;
      this.node.appendChild(this.closeEl);
      document.body.appendChild(this.style);
      document.body.appendChild(this.back);
      document.body.appendChild(this.node);
      document.addEventListener('keydown', this.onKeyDown);
      initDataAnimationTimeout = window.setTimeout(() => {
        this.back!.setAttribute('data-animation', '');
        this.node!.setAttribute('data-animation', '');
        iframe.focus();
      }, 400);
    },
    close() {
      if (this.node) {
        if (this.iframe) {
          this.node.removeChild(this.iframe);
        }
        this.onAnimationClose();
        this.node.removeAttribute('data-animation');
      }
      if (this.back) {
        this.back.removeAttribute('data-animation');
      }
      document.removeEventListener('keydown', this.onKeyDown);
    },
    delay: null,
    events: ['', 'webkit', 'moz', 'MS', 'o'].map((prefix) => (prefix ? `${prefix}TransitionEnd` : 'transitionend')),
    onAnimationClose() {
      const el = this.node!;
      if (!this.node) {
        return;
      }
      this.delay = window.setTimeout(this.animationStop, 1000);
      this.events.forEach((event) => el.addEventListener(event, this.animationStop, false));
    },
    onKeyDown(e) {
      // ESCAPE key pressed
      if (e.keyCode === 27) {
        userInfo.close();
      }
    },
    animationStop() {
      const t = userInfo;
      if (!t.node) {
        return;
      }
      if (t.delay) {
        clearTimeout(t.delay);
        t.delay = null;
      }
      t.events.forEach((event) => t.node!.removeEventListener(event, t.animationStop, false));
      return t.remove();
    },
    remove() {
      const t = userInfo;
      removeDomNode(t.node);
      removeDomNode(t.back);
      removeDomNode(t.style);
    },
  };

  function receiveMessages(event: { data?: string }): void {
    try {
      const data = typeof event.data === 'string' ? JSON.parse(event.data) : event.data;
      if (data.remarkIframeHeight) {
        iframe.style.height = `${data.remarkIframeHeight}px`;
      }

      if (data.scrollTo) {
        window.scrollTo(window.pageXOffset, data.scrollTo + iframe.getBoundingClientRect().top + window.pageYOffset);
      }

      if (Object.prototype.hasOwnProperty.call(data, 'isUserInfoShown')) {
        if (data.isUserInfoShown) {
          userInfo.init(data.user || {});
        } else {
          userInfo.close();
        }
      }

      if (data.inited) {
        postHashToIframe();
        postTitleToIframe(document.title);
      }
    } catch (e) {}
  }

  function postHashToIframe(e?: Event & { newURL: string }) {
    const hash = e ? `#${e.newURL.split('#')[1]}` : window.location.hash;

    if (hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
      if (e) e.preventDefault();

      iframe.contentWindow!.postMessage(JSON.stringify({ hash }), '*');
    }
  }

  function postTitleToIframe(title: string) {
    iframe.contentWindow!.postMessage(JSON.stringify({ title }), '*');
  }

  function postClickOutsideToIframe(e: MouseEvent) {
    if (!iframe.contains(e.target as Node)) {
      iframe.contentWindow!.postMessage(JSON.stringify({ clickOutside: true }), '*');
    }
  }

  function changeTheme(theme: Theme) {
    iframe.contentWindow!.postMessage(JSON.stringify({ theme }), '*');
  }

  function destroy() {
    if (initDataAnimationTimeout) {
      clearTimeout(initDataAnimationTimeout);
    }

    window.removeEventListener('message', receiveMessages);
    window.removeEventListener('hashchange', postHashToIframe);
    document.removeEventListener('click', postClickOutsideToIframe);

    if (titleObserver) {
      titleObserver.disconnect();
    }

    iframe.remove();
  }

  // TODO: These do not appear in Chrome DevTools
  window.REMARK42.changeTheme = changeTheme;
  window.REMARK42.destroy = () => {
    destroy();
    delete window.REMARK42.changeTheme;
    delete window.REMARK42.destroy;
  };

  return {
    changeTheme,
    destroy,
  };
}
