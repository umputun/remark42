import type { Theme, Profile } from 'common/types';

type ParentMessage = {
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

  // The size of vertical padding on body is 12px
  const bodyHeight = document.body.offsetHeight + 12;

  postMessageToParent({ height: Math.max(scrollHeight, bodyHeight) });
}
