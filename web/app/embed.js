/* eslint-disable no-console */
import { BASE_URL, NODE_ID, COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function showUserInfo(siteID, user) {
  const remarkRootId = 'remark-km423lmfdslkm34';

  const style = document.createElement('style');
  style.setAttribute('rel', 'stylesheet');
  style.setAttribute('type', 'text/css');
  style.innerHTML = `
		#${remarkRootId}-node {
			position: fixed;
			top: 0;
			right: 0;
			bottom: 0;
			width: 400px;
			transition: transform 0.4s ease-out;
			max-width: 100%;
			transform: translate(400px, 0);
			z-index: 999;
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
  document.head.appendChild(style);

  // semitransparent overlay
  const back = document.createElement('div');
  back.id = remarkRootId + '-back';
  back.onclick = () => destroy();
  document.body.appendChild(back);

  const node = document.createElement('div');
  node.id = remarkRootId + '-node';

  // close button
  const closeEl = document.createElement('button');
  closeEl.id = remarkRootId + '-close';
  closeEl.innerHTML = '&#10006;';
  closeEl.onclick = () => destroy();
  node.appendChild(closeEl);

  const queryUserInfo =
    `site_id=${encodeURIComponent(siteID)}` +
    '&page=user-info&' +
    `&id=${user.id}&name=${user.name}&picture=${user.picture || ''}&isDefaultPicture=${user.isDefaultPicture || 0}`;
  const iframe = document.createElement('iframe');
  iframe.src = `${BASE_URL}/web/iframe.html?${queryUserInfo}`;
  iframe.width = '100%';
  iframe.height = '100%';
  iframe.frameBorder = '0';
  iframe.setAttribute('allowtransparency', 'true');
  iframe.setAttribute('scrolling', 'no');
  iframe.setAttribute('horizontalscrolling', 'no');
  iframe.setAttribute('verticalscrolling', 'no');
  iframe.tabIndex = 0;
  iframe.title = 'Remark42';
  node.appendChild(iframe);

  document.body.appendChild(node);

  document.addEventListener('keydown', onKeyDown);
  setTimeout(() => {
    back.setAttribute('data-animation', '');
    node.setAttribute('data-animation', '');
    iframe.focus();
  }, 400);

  function onKeyDown(e) {
    const escapeKeyCode = 27;
    if (e.keyCode == escapeKeyCode) destroy();
  }

  function destroy() {
    document.removeEventListener('keydown', onKeyDown);

    back.removeAttribute('data-animation');
    node.removeAttribute('data-animation');
    setTimeout(() => {
      node.remove();
      back.remove();
      style.remove();
    }, 1000);
  }
}

function initNode(node, remark_config) {
  const config = {
    url: (node.dataset.url || remark_config.url || window.location.href).split('#')[0],
    site_id: node.dataset.siteId || remark_config.site_id,
  };

  if (node.dataset.maxShownComments) {
    config.max_shown_comments = node.dataset.maxShownComments;
  } else if (remark_config.max_shown_comments) {
    config.max_shown_comments = remark_config.max_shown_comments;
  }

  const query = Object.keys(config)
    .map(key => `${encodeURIComponent(key)}=${encodeURIComponent(config[key])}`)
    .join('&');

  const iframe = document.createElement('iframe');
  iframe.src = `${BASE_URL}/web/iframe.html?${query}`;
  iframe.width = '100%';
  iframe.frameBorder = '0';
  iframe.setAttribute('allowtransparency', 'true');
  iframe.setAttribute('scrolling', 'no');
  iframe.setAttribute('horizontalscrolling', 'no');
  iframe.setAttribute('verticalscrolling', 'no');
  iframe.tabIndex = 0;
  iframe.title = 'Remark42';
  iframe.style =
    'width:1px !important; min-width: 100% !important; border: none !important; overflow: hidden !important';

  node.appendChild(iframe);

  function receiveMessages(event) {
    try {
      const data = typeof event.data === 'string' ? JSON.parse(event.data) : event.data;
      if (data.remarkIframeHeight) {
        iframe.style.height = `${data.remarkIframeHeight}px`;
      }

      if (data.scrollTo) {
        window.scrollTo(window.pageXOffset, data.scrollTo + iframe.getBoundingClientRect().top + window.pageYOffset);
      }

      if (data.hasOwnProperty('isUserInfoShown') && data.isUserInfoShown) {
        showUserInfo(config.site_id, data.user || {});
      }
    } catch (e) {
      console.error(e);
    }
  }

  function postHashToIframe(e) {
    const hash = e ? `#${e.newURL.split('#')[1]}` : window.location.hash;

    if (hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
      if (e) e.preventDefault();

      iframe.contentWindow.postMessage(JSON.stringify({ hash }), '*');
    }
  }

  function postClickOutsideToIframe(e) {
    if (!iframe.contains(e.target)) {
      iframe.contentWindow.postMessage(JSON.stringify({ clickOutside: true }), '*');
    }
  }

  function destroy() {
    node.innerHTML = '';
    window.removeEventListener('message', receiveMessages);
    window.removeEventListener('hashchange', postHashToIframe);
    document.removeEventListener('click', postClickOutsideToIframe);
  }

  window.addEventListener('message', receiveMessages);
  window.addEventListener('hashchange', postHashToIframe);
  document.addEventListener('click', postClickOutsideToIframe);

  setTimeout(postHashToIframe, 1000);

  return {
    node,
    receiveMessages,
    postHashToIframe,
    postClickOutsideToIframe,
    destroy,
  };
}

function init() {
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

  if (!remark_config.selector) {
    remark_config.selector = '#' + NODE_ID;
  }

  let nodesInfo = [];

  if (typeof remark_config.selector === 'string') {
    const nodes = document.querySelectorAll(remark_config.selector);
    for (let node of nodes) {
      const info = initNode(node, remark_config);
      nodesInfo.push(info);
    }
  } else if (remark_config.selector instanceof HTMLElement) {
    const info = initNode(remark_config.selector, remark_config);
    nodesInfo.push(info);
  } else {
    console.error('TypeError: remark_config.selector should be either selector string or HTMLElement');
    return;
  }

  if (typeof remark_config.selector === 'string' && window.MutationObserver) {
    const observer = new MutationObserver(mutationList => {
      for (let record of mutationList) {
        for (let node of record.addedNodes) {
          if (node.nodeType !== 1) continue;
          let targets = [];
          if (node.matches(remark_config.selector)) targets.push(node);
          targets = targets.concat(Array.from(node.querySelectorAll(remark_config.selector)));

          for (let node of targets) {
            const info = initNode(node, remark_config);
            nodesInfo.push(info);
          }
        }

        for (let node of record.removedNodes) {
          if (node.nodeType !== 1) continue;
          let targets = [];
          if (node.matches(remark_config.selector)) targets.push(node);
          targets = targets.concat(Array.from(node.querySelectorAll(remark_config.selector)));

          for (let node of targets) {
            for (let info of nodesInfo) {
              if (node === info.node) {
                info.destroy();
                nodesInfo.splice(nodesInfo.indexOf(info), 1);
                break;
              }
            }
          }
        }
      }
    });

    observer.observe(document.body, { childList: true, subtree: true });
  }
}
