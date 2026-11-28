import { NODE_ID, COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants.config';
import { parseMessage, postMessageToIframe } from 'utils/post-message';
import { createIframe } from 'utils/create-iframe';
import type { Theme, ThemeSetting } from 'common/types';
import { closeProfile, openProfile, ownsWindow, removeProfile } from 'profile';

// marks the iframe this module owns, so a second createInstance reuses it instead of adopting
// whatever the integrator left in the root as a loading placeholder
const IFRAME_MARKER = 'data-remark42-iframe';

// what the 'auto' theme follows. Evaluated on the host page: inside the iframe the query reports the
// color-scheme of the iframe element, which is the theme the widget was given, not the visitor's
// preference
const DARK_SCHEME_QUERY = '(prefers-color-scheme: dark)';

// the widgets on the page, by the element each one is in. Every createInstance installs four
// listeners plus a title observer, all closing over that call's iframe, and `detach` takes them
// off. Given the iframe the next instance uses, it also removes its own when that is another one.
// A call into an element that holds a widget takes it over, keeping its iframe and the thread it
// shows, and a call into another element adds one. It still replaces a widget for the same `url`, one whose element has left the page,
// and the one the script mounted by itself as it loaded: a single-page site that mounts into its
// own element, or into a new one on each route, has no instance of those to destroy
const instances = new Map<Element, { url: string; auto: boolean; detach: (next?: HTMLIFrameElement) => void }>();

// the comments iframe that opened the profile overlay, which is one for the whole page
let profileOpener: HTMLIFrameElement | null = null;

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  window.REMARK42 = window.REMARK42 || {};
  window.REMARK42.createInstance = (config) => createInstance(config);

  // a single-page site loads this script once, often on a route without the comments node, and
  // waits for the event to call createInstance itself later, so it goes out even when this first
  // call throws
  try {
    createInstance(window.remark_config, true);
  } finally {
    window.dispatchEvent(new Event('REMARK42::ready'));
  }
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

// `auto` marks the widget the script mounts by itself as it loads
function createInstance(config: typeof window.remark_config, auto = false) {
  // `node` lets a single-page application mount the widget into an element it owns instead of
  // the one carrying the default id. config may be undefined here, the check below reports it
  const node = config?.node;

  if (node != null && !(node instanceof Element)) {
    throw new Error('Remark42: `node` must be a DOM element.');
  }

  const root = node ?? document.getElementById(NODE_ID);

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

  const reused = root.querySelector<HTMLIFrameElement>(`:scope > iframe[${IFRAME_MARKER}]`);
  const iframe = reused ?? createIframe(params());
  // a reused iframe goes on showing the thread it was built for, whatever this call asks for
  const shownUrl = (reused && instances.get(root)?.url) || config.url;
  iframe.setAttribute(IFRAME_MARKER, '');

  root.appendChild(iframe);

  // after the throwing validation above, so a rejected call leaves a working instance alone
  instances.forEach((instance, element) => {
    if (element === root || instance.url === shownUrl || instance.auto || !element.isConnected) {
      instance.detach(iframe);
      instances.delete(element);
    }
  });
  instances.set(root, { url: shownUrl, auto, detach });

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

  // the configuration as it goes into an iframe URL, which cannot carry a DOM element, with the
  // theme resolved
  function params() {
    const { node: _node, ...rest } = config;

    return withResolvedTheme(rest);
  }

  function handleReceiveMessage(event: MessageEvent): void {
    // only the frames this module created. window.parent is reachable by every frame on the page,
    // an advertisement among them, and the handler below resizes the widget, scrolls the page and
    // opens the profile overlay, so an unchecked sender can drive all three. A message whose
    // sender was removed before delivery has no source, and the iframe has no contentWindow once
    // the host page drops the root without destroy(), which leaves this listener attached. No
    // source on both sides is not a match
    if (event.source === null || (event.source !== iframe.contentWindow && !ownsWindow(event.source))) {
      return;
    }
    // the profile overlay is one for the page and talks to the widget that opened it; with every
    // widget acting on its sign-out, each would end the session again
    if (event.source !== iframe.contentWindow && profileOpener !== iframe) {
      return;
    }

    const data = parseMessage(event);

    if (typeof data.height === 'number') {
      iframe.style.height = `${data.height}px`;
    }

    if (typeof data.scrollTo === 'number') {
      const { scrollTo } = data;
      const scroll = () => {
        // destroy() may have removed the iframe in the frames this waits for
        if (iframe.isConnected) {
          window.scrollTo(window.pageXOffset, scrollTo + iframe.getBoundingClientRect().top + window.pageYOffset);
        }
      };

      // the frame posts its new height just before this. Scrolling in the frame that applies it
      // counts everything below the iframe as shifted once the scroll brings it into view, the
      // page footer when the comment is near the end, so one frame is drawn at the new height
      // first. A hidden tab draws no frames, and a link opened in the background scrolls at once
      if (document.visibilityState === 'hidden') {
        scroll();
      } else {
        requestAnimationFrame(() => requestAnimationFrame(scroll));
      }
    }

    if (typeof data.profile === 'object') {
      if (data.profile === null) {
        closeProfile();
      } else {
        profileOpener = iframe;
        openProfile({ ...params(), ...data.profile });
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

  function detach(next?: HTMLIFrameElement) {
    if (next && next !== iframe) {
      iframe.remove();
      // the widget taking this one's place takes over the profile it opened
      if (profileOpener === iframe) {
        profileOpener = next;
      }
    }

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
    if (instances.get(root!)?.detach === detach) {
      instances.delete(root!);
    }

    iframe.remove();
    // the overlay belongs to the widget that opened it; another widget's goes on
    if (profileOpener === iframe) {
      profileOpener = null;
      removeProfile();
    }
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
