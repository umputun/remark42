import { User } from 'common/types';

export type Message =
  | { inited: true }
  | {
      isUserInfoShown: true;
      user: User;
    }
  | { isUserInfoShown: false }
  | { scrollTo: number }
  | { remarkIframeHeight: number };

/**
 * Sends message to parent window
 *
 * @returns request success of fail
 */
export default function postMessage(data: Message): boolean {
  if (!window.parent || window.parent === window) return false;
  window.parent.postMessage(JSON.stringify(data), '*');
  return true;
}
