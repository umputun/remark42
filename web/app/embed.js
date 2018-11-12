/* eslint-disable */
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

  const styles = document.createElement('style');
  styles.innerHTML = `
    #${NODE_ID} .preloader {
      width: 60px;
      text-align: center;
      font-size: 0;
    }

    #${NODE_ID} .preloader_view_init {
      margin: 0 auto;
    }

    #${NODE_ID} .preloader__bounce {
      display: inline-block;
      width: 10px;
      height: 10px;
      margin-right: 3px;
      background-color: #333;
      border-radius: 100%;
      animation: remark42preloaderBounce 1.4s infinite ease-in-out both;
    }

    #${NODE_ID} .preloader__bounce:first-child {
      animation-delay: -.32s;
    }

    #${NODE_ID} .preloader__bounce:nth-child(2) {
      animation-delay: -.16s;
    }

    #${NODE_ID} .preloader__bounce:last-child {
      margin-right: 0;
    }

    @keyframes remark42preloaderBounce {
      0% {
        transform: scale(0);
      }

      40% {
        transform: scale(1);
      }

      80%, 100% {
        transform: scale(0);
      }
    }

    #${NODE_ID} :focus:not(.focus-visible):not(.button) {
      outline: none;
    }
  `;
  document.getElementsByTagName('head')[0].appendChild(styles);

  node.innerHTML = `
    <div class="preloader preloader_view_init">
      <div class="preloader__bounce"></div>
      <div class="preloader__bounce"></div>
      <div class="preloader__bounce"></div>
    </div>
  `;

  const remarkStyles = document.createElement('link');
  remarkStyles.setAttribute('rel', 'stylesheet');
  remarkStyles.setAttribute('href', `${BASE_URL}/web/remark.css`);
  document.getElementsByTagName('head')[0].appendChild(remarkStyles);

  const remarkScripts = document.createElement('script');
  remarkScripts.setAttribute('src', `${BASE_URL}/web/remark.js`);
  document.getElementsByTagName('head')[0].appendChild(remarkScripts);

  // const iframe = node.getElementsByTagName('iframe')[0];

  // window.addEventListener('message', receiveMessages);
  // window.addEventListener('hashchange', postHashToIframe);
  // document.addEventListener('click', postClickOutsideToIframe);
  // setTimeout(postHashToIframe, 1000);

  // const remarkRootId = 'remark-km423lmfdslkm34';
  // const userInfo = {
  //   node: null,
  //   back: null,
  //   closeEl: null,
  //   iframe: null,
  //   style: null,
  //   init(user) {
  //     this.animationStop();
  //     if (!this.style) {
  //       this.style = document.createElement('style');
  //       this.style.setAttribute('rel', 'stylesheet');
  //       this.style.setAttribute('type', 'text/css');
  //       this.style.innerHTML = `
  //         #${remarkRootId}-node {
  //           position: fixed;
  //           top: 0;
  //           right: 0;
  //           bottom: 0;
  //           width: 400px;
  //           transition: transform 0.4s ease-out;
  //           max-width: 100%;
  //           transform: translate(400px, 0);
  //         }
  //         #${remarkRootId}-node[data-animation] {
  //           transform: translate(0, 0);
  //         }
  //         #${remarkRootId}-back {
  //           position: fixed;
  //           top: 0;
  //           left: 0;
  //           right: 0;
  //           bottom: 0;
  //           background: rgba(0,0,0,0.7);
  //           opacity: 0;
  //           transition: opacity 0.4s ease-out;
  //         }
  //         #${remarkRootId}-back[data-animation] {
  //           opacity: 1;
  //         }
  //         #${remarkRootId}-close {
  //           top: 0px;
  //           right: 400px;
  //           position: absolute;
  //           text-align: center;
  //           font-size: 25px;
  //           cursor: pointer;
  //           color: white;
  //           border-color: transparent;
  //           border-width: 0;
  //           padding: 0;
  //           margin-right: 4px;
  //           background-color: transparent;
  //         }
  //         @media all and (max-width: 430px) {
  //           #${remarkRootId}-close {
  //             right: 0px;
  //             font-size: 20px;
  //             color: black;
  //           }
  //         }
  //       `;
  //     }
  //     if (!this.node) {
  //       this.node = document.createElement('div');
  //       this.node.id = remarkRootId + '-node';
  //     }
  //     if (!this.back) {
  //       this.back = document.createElement('div');
  //       this.back.id = remarkRootId + '-back';
  //       this.back.onclick = () => this.close();
  //     }
  //     if (!this.closeEl) {
  //       this.closeEl = document.createElement('button');
  //       this.closeEl.id = remarkRootId + '-close';
  //       this.closeEl.innerHTML = '&#10006;';
  //       this.closeEl.onclick = () => this.close();
  //     }
  //     const queryUserInfo =
  //       query +
  //       '&page=user-info&' +
  //       `&id=${user.id}&name=${user.name}&picture=${user.picture || ''}&isDefaultPicture=${user.isDefaultPicture || 0}`;
  //     this.node.innerHTML = `
  //     <iframe
  //       src="${BASE_URL}/web/iframe.html?${queryUserInfo}"
  //       width="100%"
  //       height="100%"
  //       frameborder="0"
  //       allowtransparency="true"
  //       scrolling="no"
  //       tabindex="0"
  //       title="Remark42"
  //       verticalscrolling="no"
  //       horizontalscrolling="no"
  //     />`;
  //     this.iframe = this.node.querySelector('iframe');
  //     this.node.appendChild(this.closeEl);
  //     document.body.appendChild(this.style);
  //     document.body.appendChild(this.back);
  //     document.body.appendChild(this.node);
  //     document.addEventListener('keydown', this.onKeyDown);
  //     setTimeout(() => {
  //       this.back.setAttribute('data-animation', '');
  //       this.node.setAttribute('data-animation', '');
  //       this.iframe.focus();
  //     }, 400);
  //   },
  //   close() {
  //     if (this.node) {
  //       this.onAnimationClose();
  //       this.node.removeAttribute('data-animation');
  //     }
  //     if (this.back) {
  //       this.back.removeAttribute('data-animation');
  //     }
  //     document.removeEventListener('keydown', this.onKeyDown);
  //   },
  //   delay: null,
  //   events: ['', 'webkit', 'moz', 'MS', 'o'].map(prefix => (prefix ? `${prefix}TransitionEnd` : 'transitionend')),
  //   onAnimationClose() {
  //     const el = this.node;
  //     if (!this.node) {
  //       return;
  //     }
  //     this.delay = setTimeout(this.animationStop, 1000);
  //     this.events.forEach(event => el.addEventListener(event, this.animationStop, false));
  //   },
  //   onKeyDown(e) {
  //     // ESCAPE key pressed
  //     if (e.keyCode == 27) {
  //       userInfo.close();
  //     }
  //   },
  //   animationStop() {
  //     const t = userInfo;
  //     if (!t.node) {
  //       return;
  //     }
  //     if (t.delay) {
  //       clearTimeout(t.delay);
  //       t.delay = null;
  //     }
  //     t.events.forEach(event => t.node.removeEventListener(event, t.animationStop, false));
  //     return t.remove();
  //   },
  //   remove() {
  //     const t = userInfo;
  //     t.node && t.node.remove();
  //     t.back && t.back.remove();
  //     t.style && t.style.remove();
  //   },
  // };

  // function receiveMessages(event) {
  //   try {
  //     const data = typeof event.data === 'string' ? JSON.parse(event.data) : event.data;
  //     if (data.remarkIframeHeight) {
  //       iframe.style.height = `${data.remarkIframeHeight}px`;
  //     }
  //
  //     if (data.scrollTo) {
  //       window.scrollTo(window.pageXOffset, data.scrollTo + iframe.getBoundingClientRect().top + window.pageYOffset);
  //     }
  //
  //     if (data.hasOwnProperty('isUserInfoShown')) {
  //       if (data.isUserInfoShown) {
  //         userInfo.init(data.user || {});
  //       } else {
  //         userInfo.close();
  //       }
  //     }
  //   } catch (e) {}
  // }

  // function postHashToIframe(e) {
  //   const hash = e ? `#${e.newURL.split('#')[1]}` : window.location.hash;
  //
  //   if (hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
  //     if (e) e.preventDefault();
  //
  //     iframe.contentWindow.postMessage(JSON.stringify({ hash }), '*');
  //   }
  // }
  //
  // function postClickOutsideToIframe(e) {
  //   if (!iframe.contains(e.target)) {
  //     iframe.contentWindow.postMessage(JSON.stringify({ clickOutside: true }), '*');
  //   }
  // }
}
