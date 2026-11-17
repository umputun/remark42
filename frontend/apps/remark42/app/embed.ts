import { NODE_ID, COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants.config';
import { parseMessage, postMessageToIframe } from 'utils/post-message';
import { createIframe } from 'utils/create-iframe';
import type { Theme, ThemeSetting } from 'common/types';
import { closeProfile, openProfile, ownsWindow } from 'profile';

// marks the iframe this module owns, so a second createInstance reuses it instead of adopting
// whatever the integrator left in the root as a loading placeholder
const IFRAME_MARKER = 'data-remark42-iframe';

// what the 'auto' theme follows. Evaluated on the host page: inside the iframe the query reports the
// color-scheme of the iframe element, which is the theme the widget was given, not the visitor's
// preference
const DARK_SCHEME_QUERY = '(prefers-color-scheme: dark)';

// detaches the listeners of whichever instance is current. every createInstance installs four of
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

// resolves 'auto' to the theme the visitor's system prefers. The widget itself only knows light and dark
function resolveTheme(theme: ThemeSetting): Theme {
  if (theme !== 'auto') {
    return theme;
  }

  return window.matchMedia?.(DARK_SCHEME_QUERY).matches ? 'dark' : 'light';
}

// the config with 'auto' resolved, for building an iframe: its URL carries the theme the document
// starts in, and the document only knows light and dark
function withResolvedTheme(config: typeof window.remark_config) {
  return config.theme ? { ...config, theme: resolveTheme(config.theme) } : config;
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
  // absent where the browser has no matchMedia, and then 'auto' stays light
  const darkScheme = window.matchMedia?.(DARK_SCHEME_QUERY);

  config.url = (config.url || `${window.location.origin}${window.location.pathname}`).split('#')[0];

  const iframe =
    root.querySelector<HTMLIFrameElement>(`:scope > iframe[${IFRAME_MARKER}]`) ??
    createIframe(withResolvedTheme(config));
  iframe.setAttribute(IFRAME_MARKER, '');

  root.appendChild(iframe);

  // after the throwing validation above, so a rejected call leaves a working instance alone
  detachCurrentInstance?.();
  detachCurrentInstance = detach;

  window.addEventListener('message', handleReceiveMessage);
  window.addEventListener('hashchange', handleHashChange);
  document.addEventListener('click', postClickOutsideToIframe);
  darkScheme?.addEventListener('change', handleSchemeChange);

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
        openProfile({ ...withResolvedTheme(config), ...data.profile });
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
      // a changeTheme made before the document loaded was posted to a frame with no one listening,
      // and the iframe URL only carries the theme it was built with
      applyTheme();
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

  // sets the configured theme, with 'auto' resolved, on the iframe element and posts it to the
  // document inside, which matches its own color-scheme to it
  function applyTheme() {
    if (!config.theme) {
      return;
    }

    const theme = resolveTheme(config.theme);

    iframe.style.colorScheme = theme;
    postMessageToIframe(iframe, { theme });
  }

  function handleSchemeChange() {
    if (config.theme === 'auto') {
      applyTheme();
    }
  }

  function postClickOutsideToIframe(evt: MouseEvent) {
    if (iframe.contains(evt.target as Node)) {
      return;
    }
    postMessageToIframe(iframe, { clickOutside: true });
  }

  function changeTheme(theme: ThemeSetting) {
    config.theme = theme;
    window.remark_config.theme = theme;
    applyTheme();
  }

  function detach() {
    window.removeEventListener('message', handleReceiveMessage);
    window.removeEventListener('hashchange', handleHashChange);
    document.removeEventListener('click', postClickOutsideToIframe);
    darkScheme?.removeEventListener('change', handleSchemeChange);

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
