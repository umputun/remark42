import { NODE_ID, COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants.config';
import { parseMessage, postMessageToIframe } from 'utils/post-message';
import { createIframe } from 'utils/create-iframe';
import type { Theme } from 'common/types';
import { closeProfile, openProfile, ownsWindow } from 'profile';

// marks the iframe this module owns, so a second createInstance reuses it instead of adopting
// whatever the integrator left in the root as a loading placeholder
const IFRAME_MARKER = 'data-remark42-iframe';

// detaches the listeners of whichever instance is current. every createInstance installs three of
// them plus a title observer, all closing over that call's iframe, while the call reuses an
// existing iframe rather than building one. Without this a second call leaves the first set
// attached for good, since destroy() can only reach the newest closure
let detachCurrentInstance: (() => void) | null = null;

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
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

  let titleObserver: MutationObserver | null = null;

  config.url = (config.url || `${window.location.origin}${window.location.pathname}`).split('#')[0];

  const iframe = root.querySelector<HTMLIFrameElement>(`:scope > iframe[${IFRAME_MARKER}]`) ?? createIframe(config);
  iframe.setAttribute(IFRAME_MARKER, '');

  root.appendChild(iframe);

  // after the throwing validation above, so a rejected call leaves a working instance alone
  detachCurrentInstance?.();
  detachCurrentInstance = detach;

  window.addEventListener('message', handleReceiveMessage);
  window.addEventListener('hashchange', handleHashChange);
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

  function handleReceiveMessage(event: MessageEvent): void {
    // only the frames this module created. window.parent is reachable by every frame on the page,
    // an advertisement among them, and the handler below resizes the widget, scrolls the page and
    // opens the profile overlay, so an unchecked sender can drive all three
    if (event.source !== iframe.contentWindow && !ownsWindow(event.source)) {
      return;
    }

    const data = parseMessage(event);

    if (typeof data.height === 'number') {
      iframe.style.height = `${data.height}px`;
    }

    if (typeof data.scrollTo === 'number') {
      window.scrollTo(window.pageXOffset, data.scrollTo + iframe.getBoundingClientRect().top + window.pageYOffset);
    }

    if (typeof data.profile === 'object') {
      if (data.profile === null) {
        closeProfile();
      } else {
        openProfile({ ...config, ...data.profile });
      }
    }

    if (data.signout === true) {
      postMessageToIframe(iframe, { signout: true });
    }

    if (data.inited === true) {
      // remove placeholder content added by the user before comments loaded
      Array.from(root!.childNodes)
        .filter((node) => node !== iframe)
        .forEach((node) => node.remove());

      postHashToIframe();
      postTitleToIframe(document.title);
    }
  }

  function postHashToIframe(hash = window.location.hash) {
    if (!hash.startsWith(`#${COMMENT_NODE_CLASSNAME_PREFIX}`)) {
      return;
    }

    postMessageToIframe(iframe, { hash });
  }

  function handleHashChange(evt: HashChangeEvent) {
    const url = new URL(evt.newURL);

    postHashToIframe(url.hash);
  }

  function postTitleToIframe(title: string) {
    postMessageToIframe(iframe, { title });
  }

  function postClickOutsideToIframe(evt: MouseEvent) {
    if (iframe.contains(evt.target as Node)) {
      return;
    }
    postMessageToIframe(iframe, { clickOutside: true });
  }

  function changeTheme(theme: Theme) {
    window.remark_config.theme = theme;
    iframe.style.colorScheme = theme;
    postMessageToIframe(iframe, { theme });
  }

  function detach() {
    window.removeEventListener('message', handleReceiveMessage);
    window.removeEventListener('hashchange', handleHashChange);
    document.removeEventListener('click', postClickOutsideToIframe);

    if (titleObserver) {
      titleObserver.disconnect();
      // Allow browser to drop observer and iframe captured in callback
      // to prevent attempts to send messages to detached frame
      titleObserver = null;
    }
  }

  function destroy() {
    detach();

    // a later createInstance must not detach an instance that is already gone
    if (detachCurrentInstance === detach) {
      detachCurrentInstance = null;
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
