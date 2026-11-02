import type { Theme, Profile } from 'common/types';

type ParentMessage = {
  /** the iframe document bootstrapped: drives placeholder removal and iframe reveal */
  inited?: true;
  scrollTo?: number;
  height?: number;
  signout?: true;
  profile?: Profile | null;
};

type ChildMessage = {
  clickOutside?: true;
  hash?: string;
  title?: string;
  theme?: Theme;
  signout?: true;
};

type AllMessages = ChildMessage & ParentMessage;

/**
 * Sends message to parent window
 *
 * @returns request success of fail
 */
export function postMessageToParent(data: ParentMessage): boolean {
  if (!window.parent || window.parent === window) {
    return false;
  }

  window.parent.postMessage(data, '*');
  return true;
}

/**
 * Sends message to target iframe
 *
 * @param target iframe to send data
 * @param data that will be sent to iframe
 * @returns request success of fail
 */
export function postMessageToIframe(target: HTMLIFrameElement, data: ChildMessage): boolean {
  if (!target?.contentWindow) {
    return false;
  }

  target.contentWindow.postMessage(data, '*');
  return true;
}

/**
 * Whether a message reaching the widget document came from the page embedding it.
 *
 * The embed script is the only legitimate sender, and it always posts from the parent. Browser
 * extensions post their own messages into the frame, and so can any window holding a reference to
 * it, which matters because the child acts on `signout` and `theme`. The origin cannot be checked
 * instead: the host page is whatever site embeds the widget, so there is no fixed value to compare
 * against, and `ALLOWED_HOSTS` is enforced server side through `frame-ancestors` rather than here.
 *
 * A widget opened directly rather than embedded has `window.parent === window`, so its own messages
 * still pass, which is what the counter and last-comments pages rely on.
 */
export function isFromParent(evt: MessageEvent): boolean {
  return evt.source === window.parent;
}

/**
 * Parses data from post message that was received in iframe
 *
 * @param evt post message event
 * @returns
 */
export function parseMessage({ data }: MessageEvent): AllMessages {
  if (typeof data !== 'object' || data === null || Array.isArray(data)) {
    return {};
  }

  return data as AllMessages;
}

/**
 * Sends message to parent window with height of iframe
 *
 * @param dropdown provide dropdown element if present on screen
 */
export function updateIframeHeight(dropdown?: HTMLElement) {
  let scrollHeight = 0;

  // If dropdown is present on screen, we need to calculate size according to it size since it's positioned absolutely
  if (dropdown) {
    const { top } = dropdown.getBoundingClientRect();
    // The size of shadow under the dropdown is 20px
    scrollHeight = window.scrollY + Math.abs(top) + dropdown.scrollHeight + 20;
  }

  // offsetHeight already covers padding and border, so it is the full document height
  const bodyHeight = document.body.offsetHeight;

  postMessageToParent({ height: Math.max(scrollHeight, bodyHeight) });
}
