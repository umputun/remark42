import { getUser } from 'common/api';
import { User } from 'common/types';

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

  return new Promise((resolve, reject) => {
    function unsubscribe() {
      document.removeEventListener('visibilitychange', handleWindowVisibilityChange);
      window.removeEventListener('focus', handleWindowVisibilityChange);
      subscribed = false;
      clearTimeout(timeout);
    }

    async function handleWindowVisibilityChange() {
      if (!document.hasFocus() || document.hidden || !authWindow?.closed) {
        return;
      }

      const user = await getUser();

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

    setTimeout(() => {
      reject();
    }, 5 * 60 * 1000);

    document.addEventListener('visibilitychange', handleWindowVisibilityChange);
    window.addEventListener('focus', handleWindowVisibilityChange);
  });
}
