import type { Theme, UserInfo } from 'common/types';

export type ParentMessage = {
  inited?: true;
  scrollTo?: number;
  remarkIframeHeight?: number;
} & (
  | { isUserInfoShown: true; user: UserInfo }
  | { isUserInfoShown: false; user?: never }
  | { isUserInfoShown?: never; user?: never }
);

export type ChildMessage = {
  theme?: Theme;
};

type AllMessages = ParentMessage & ChildMessage;

/**
 * Sends message to parent window
 *
 * @returns request success of fail
 */
export function postMessage(data: AllMessages): boolean {
  if (!window.parent || window.parent === window) return false;
  window.parent.postMessage(data, '*');
  return true;
}

/**
 * Parses data from post message that was received in iframe
 *
 * @param evt post message event
 * @returns
 */
export function parseMessage<T>({ data }: MessageEvent<T>): T {
  if (typeof data !== 'object' || data === null || Array.isArray(data)) {
    return {} as T;
  }

  return data as T;
}
