import type { User } from 'common/types';

import { authFetcher } from 'common/fetcher';
import { siteId } from 'common/settings';
import { getUser } from 'common/api';

const EMAIL_SIGNIN_ENDPOINT = '/email/login';
const TELEGRAM_SIGNIN_ENDPOINT = '/telegram/login';

export function anonymousSignin(user: string): Promise<User> {
  return authFetcher.get<User>('/anonymous/login', { user, aud: siteId });
}

/**
 * First step of two of `email` authorization
 */
export function emailSignin(email: string, username: string): Promise<unknown> {
  return authFetcher.get(EMAIL_SIGNIN_ENDPOINT, { address: email, user: username });
}

/**
 * Second step of two of `email` authorization
 */
export function verifyEmailSignin(token: string): Promise<User> {
  return authFetcher.get(EMAIL_SIGNIN_ENDPOINT, { token });
}

/**
 * Performs await of auth from oauth providers
 */
let subscribed = false;
let timeout: NodeJS.Timeout;
let authWindow: Window | null = null;

/**
 * Set waiting state and tries to revalidate `user` when oauth tab is closed
 */
export function oauthSignin(url: string): Promise<User | null> {
  authWindow = window.open(url);

  if (subscribed) {
    return Promise.resolve(null);
  }

  subscribed = true;

  return new Promise((resolve, reject) => {
    // closure-local rather than the module-level `subscribed`, because a second oauthSignin can
    // start while this one's getUser is still resolving and would clear a flag it does not own
    let stopped = false;

    function unsubscribe() {
      stopped = true;
      document.removeEventListener('visibilitychange', handleWindowVisibilityChange);
      window.removeEventListener('focus', handleWindowVisibilityChange);
      subscribed = false;
      clearTimeout(timeout);
      clearTimeout(giveUp);
    }

    async function handleWindowVisibilityChange() {
      if (!document.hasFocus() || document.hidden || !authWindow?.closed) {
        return;
      }

      const user = await getUser();

      // the request was in flight when the deadline passed, so there is nothing left to clear the
      // retry this would otherwise schedule
      if (stopped) {
        return;
      }

      clearTimeout(timeout);

      if (user === null) {
        // Retry after 1 min if current attempt unsuccessful
        timeout = setTimeout(() => {
          handleWindowVisibilityChange();
        }, 60 * 1000);

        return null;
      }

      resolve(user);
      unsubscribe();
    }

    // giving up has to tear the subscription down as well. the retry above reschedules itself
    // for as long as getUser keeps returning null, which is the permanent state whenever the
    // widget is embedded cross-domain and the cookie never becomes readable, so leaving the
    // listeners attached leaves a poll of /auth/user running for the life of the page against
    // a route limited to 2 req/s
    const giveUp = setTimeout(
      () => {
        unsubscribe();
        reject(new Error('Timed out waiting for the authorization window'));
      },
      5 * 60 * 1000
    );

    document.addEventListener('visibilitychange', handleWindowVisibilityChange);
    window.addEventListener('focus', handleWindowVisibilityChange);
  });
}

/**
 * First step of two of `telegram` authorization
 */
export function getTelegramSigninParams(): Promise<{
  bot: string;
  token: string;
}> {
  return authFetcher.get(TELEGRAM_SIGNIN_ENDPOINT);
}

/**
 * Second step of two of `telegram` authorization
 */
export function verifyTelegramSignin(token: string): Promise<User> {
  return authFetcher.get(TELEGRAM_SIGNIN_ENDPOINT, { token });
}

export function logout(): Promise<void> {
  return authFetcher.get('/logout');
}
