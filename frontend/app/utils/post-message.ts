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
 * @param data that will be send to iframe
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
    return {} as AllMessages;
  }

  return data as AllMessages;
}
