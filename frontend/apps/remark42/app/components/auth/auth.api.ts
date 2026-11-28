import type { User } from 'common/types';

import { authFetcher, dropSession, forgetToken, probeSignedInUser } from 'common/fetcher';
import { siteId } from 'common/settings';
import { BASE_URL } from 'common/constants.config';
import { getSignedInUser, getUser } from 'common/api';
import { errorMessages, RequestError } from 'utils/errorUtils';

import { handoffStartURL, handoffTxn, HANDOFF_ACK, POPUP_BLOCKED, takeHandoff } from './auth.handoff';

export { POPUP_BLOCKED };

const EMAIL_SIGNIN_ENDPOINT = '/email/login';
const TELEGRAM_SIGNIN_ENDPOINT = '/telegram/login';

/**
 * The user a sign-in endpoint answers with is the auth library's own shape, which carries the
 * admin flag under `attrs` rather than as `admin`, so storing it made an admin a regular user
 * until the next page load (#2205). Each sign-in loads the user from the API instead. It goes to
 * `/user` directly rather than through `getUser`, whose `/auth/status` probe has nothing to tell
 * straight after a sign-in has succeeded
 */
async function signin(endpoint: string, query: Record<string, string>): Promise<User> {
  await authFetcher.get(endpoint, query);
  return getSignedInUser();
}

export function anonymousSignin(user: string): Promise<User> {
  return signin('/anonymous/login', { user, aud: siteId });
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
  return signin(EMAIL_SIGNIN_ENDPOINT, { token });
}

/**
 * Performs await of auth from oauth providers
 */
let subscribed = false;
let timeout: NodeJS.Timeout;
let authWindow: Window | null = null;

/**
 * One click on a provider. With a verifier the popup opens on the handoff start page, and once that
 * page has told this frame its transaction, the transaction decides the outcome. A second click makes
 * a new attempt and leaves the earlier one
 */
type Attempt = { popup: Window; verifier: string | null; txn: string | null; handedOver: boolean };
let attempt: Attempt | null = null;

/** How long a handoff status request that failed on the way waits before it is sent again */
export const HANDOFF_RETRY_DELAY_MS = 2000;

/**
 * The shortest time between two status requests. The backend holds a pending one for a while, so
 * this only matters for an answer that comes back at once, as one through a proxy that does not
 * wait might, which would otherwise be asked again as fast as it answers
 */
export const HANDOFF_POLL_MIN_MS = 1000;

/** The start page gave the sign-in up before this frame answered, and went to the ordinary login */
class HandoffCanceled extends Error {}

/**
 * Whether the request did not get through, and can be sent again: a network failure, which every
 * browser but Chrome throws as a plain TypeError, the rate limit, a server or gateway error, or an
 * error page that is not the backend's own
 */
function isTransient(e: unknown): boolean {
  if (e instanceof TypeError) {
    return true;
  }
  if (!(e instanceof RequestError)) {
    return false;
  }
  if (e.code === 'fetch-error') {
    return true;
  }
  const status = httpStatus(e);
  return status === 429 || (status !== undefined && status >= 500) || (e.status === undefined && e.code === 0);
}

/** The HTTP status of a failed request, which the code stands for when the response named none */
function httpStatus(e: RequestError): number | undefined {
  return e.status ?? (typeof e.code === 'number' ? e.code : undefined);
}

/** How many times a request of the handoff that did not get through is sent again */
export const HANDOFF_RETRIES = 4;

type HandoffStatus = { status: 'pending' | 'done' | 'canceled'; user?: string };

/**
 * A call to the handoff routes. They answer the frame on its own origin and need none of what the
 * fetcher adds, and their status code is the answer, which the fetcher folds away for most codes.
 * A status that is not 2xx rejects with a RequestError carrying it
 */
async function handoff<T>(call: 'status' | 'redeem' | 'cancel', body: { txn: string | null; verifier: string | null }) {
  const res = await fetch(`${BASE_URL}/auth/handoff/${call}`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    // the status is what the handoff goes by; the reader sees the catalogue's general message
    throw new RequestError(errorMessages[0].defaultMessage as string, 0, res.status);
  }
  return (res.status === 204 ? null : await res.json()) as T;
}

const pause = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/**
 * Waits for the provider to bring the reader back, then signs this frame in as the account the
 * sign-in recorded. The session the frame sees already is used when it is that account: on the same
 * site, and wherever the browser sends remark42's own cookies to the frame. Otherwise the sign-in
 * is redeemed for a session of the frame's own. An earlier session, of whoever, never counts
 */
async function completeHandoff(a: Attempt, live: () => boolean): Promise<User> {
  const body = { txn: a.txn, verifier: a.verifier };
  const left = () => new Error('Sign-in left');
  let status: HandoffStatus | null = null;

  while (live()) {
    const asked = Date.now();
    try {
      status = await handoff<HandoffStatus>('status', body);
    } catch (e) {
      if (!isTransient(e)) {
        throw e;
      }
      await pause(HANDOFF_RETRY_DELAY_MS);
      continue;
    }
    if (status.status !== 'pending') {
      break;
    }
    await pause(Math.max(0, asked + HANDOFF_POLL_MIN_MS - Date.now()));
  }
  if (!live() || !status) {
    throw left();
  }
  if (status.status === 'canceled') {
    throw new HandoffCanceled();
  }
  const expected = status.user;

  // only a refusal says the frame has no session for this site: 401 for none, 403 for one of
  // another site served from this host. A request that did not get through says nothing, and
  // redeeming on it would move a frame that shares the site's session onto one of its own
  async function seenUser(): Promise<User | null> {
    for (let attempt = 0; ; attempt++) {
      let failure: unknown;
      try {
        const { status, user } = await probeSignedInUser<User>();
        if (user) {
          return user;
        }
        if (status === 401 || status === 403) {
          return null;
        }
        failure = new RequestError(errorMessages[0].defaultMessage as string, 0, status);
      } catch (e) {
        failure = e;
      }
      if (!isTransient(failure) || attempt >= HANDOFF_RETRIES || !live()) {
        throw failure;
      }
      await pause(HANDOFF_RETRY_DELAY_MS);
    }
  }

  const seen = await seenUser();
  if (seen && seen.id === expected) {
    // the frame goes on with the session its cookies carry; a token in memory is read ahead of
    // them and may be a session of another site
    forgetToken();
    handoff('cancel', body).catch(() => undefined);
    return seen;
  }
  // a second click while the account was being looked up makes a new attempt, which this one
  // leaves to it instead of putting its own session in place
  if (!live()) {
    throw left();
  }

  try {
    // the rate limit answers ahead of the handler, so the transaction is still there to redeem
    for (let attempt = 0; ; attempt++) {
      try {
        await handoff('redeem', body);
        break;
      } catch (e) {
        if (!(e instanceof RequestError && e.status === 429) || attempt >= HANDOFF_RETRIES || !live()) {
          throw e;
        }
        await pause(HANDOFF_RETRY_DELAY_MS);
        if (!live()) {
          throw left();
        }
      }
    }
  } catch (e) {
    // the backend saw this frame send remark42's own session for the account, and gives no session
    // of the frame's own where that one is seen: the frame takes it
    if (!(e instanceof RequestError && e.status === 409)) {
      throw e;
    }
    // a token in memory is read ahead of the cookie, and may be a session of another site
    forgetToken();
    const again = await seenUser();
    if (!again || again.id !== expected) {
      throw e;
    }
    handoff('cancel', body).catch(() => undefined);
    return again;
  }
  // the token in memory, if any, is the other session's
  forgetToken();
  const user = await seenUser();
  if (!user || user.id !== expected) {
    throw new RequestError(errorMessages[0].defaultMessage as string, 0);
  }
  return user;
}

/**
 * Set waiting state and tries to revalidate `user` when oauth tab is closed
 */
export function oauthSignin(url: string): Promise<User | null> {
  const pair = takeHandoff();
  const start = pair ? handoffStartURL(url, pair.challenge) : null;
  const opened = window.open(start ?? url);

  // a browser that blocks the window says so with null, or with a window closed from the start
  if (!opened || opened.closed) {
    return Promise.reject(new RequestError('The sign-in window was blocked', POPUP_BLOCKED));
  }

  authWindow = opened;
  attempt = { popup: opened, verifier: start && pair ? pair.verifier : null, txn: null, handedOver: false };

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
      window.removeEventListener('message', handleMessage);
      subscribed = false;
      clearTimeout(timeout);
      clearTimeout(giveUp);
    }

    function handleMessage(evt: MessageEvent) {
      const a = attempt;
      if (!a || !a.verifier || a.txn || evt.source !== a.popup || evt.origin !== window.location.origin) {
        return;
      }
      const txn = handoffTxn(evt.data);
      if (!txn) {
        return;
      }
      a.txn = txn;
      a.handedOver = true;
      a.popup.postMessage({ type: HANDOFF_ACK, txn }, window.location.origin);

      const live = () => !stopped && attempt === a;
      completeHandoff(a, live).then(
        (user) => {
          if (live()) {
            // only now: the popup may still be storing the callback's cookies until the sign-in
            // is through, and the done page closes itself anyway
            try {
              a.popup.close();
            } catch (err) {
              // the provider may have cut the popup off from this frame
            }
            resolve(user);
            unsubscribe();
          }
        },
        (e) => {
          if (!live()) {
            handoff('cancel', { txn: a.txn, verifier: a.verifier }).catch(() => undefined);
            return;
          }
          if (e instanceof HandoffCanceled) {
            // the popup went on to the ordinary login without this frame, which waits for it as
            // before. The popup may have closed already, while this frame was not looking
            a.handedOver = false;
            handleWindowVisibilityChange();
            return;
          }
          // the panel shows a message for a RequestError's code. A request that never got an answer
          // gets the one the fetcher gives it, and anything else, such as a proxy's page that is not
          // JSON, the general one
          reject(
            e instanceof RequestError
              ? e
              : e instanceof TypeError
                ? new RequestError(errorMessages['fetch-error'].defaultMessage as string, 'fetch-error')
                : new RequestError(errorMessages[0].defaultMessage as string, 0)
          );
          unsubscribe();
        }
      );
    }

    async function handleWindowVisibilityChange() {
      // once the popup has handed the sign-in over, the sign-in alone says how it ended
      if (attempt?.handedOver) {
        return;
      }

      if (!document.hasFocus() || document.hidden || !authWindow?.closed) {
        return;
      }

      const user = await getUser();

      // the request was in flight when the deadline passed, so there is nothing left to clear the
      // retry this would otherwise schedule
      if (stopped || attempt?.handedOver) {
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
        // with a code, which the panel shows in the reader's language
        reject(new RequestError('Timed out waiting for the authorization window', 0));
      },
      5 * 60 * 1000
    );

    document.addEventListener('visibilitychange', handleWindowVisibilityChange);
    window.addEventListener('focus', handleWindowVisibilityChange);
    window.addEventListener('message', handleMessage);
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
  return signin(TELEGRAM_SIGNIN_ENDPOINT, { token });
}

/**
 * How long a sign-out request that failed waits before it is sent once more, in milliseconds. The
 * backend takes two requests a second on its auth routes from one address and answers the rest
 * with 429
 */
export const LOGOUT_RETRY_DELAY_MS = 1000;

/** A sign-out the backend refused is its answer. Anything else did not reach the logout handler */
function isRefused(e: unknown): boolean {
  return e instanceof RequestError && (e.code === 401 || e.code === 403);
}

/**
 * Signs out, and forgets the session locally whatever the server answers. With the JWT sent as a
 * header, an expired token is refused with 403, and a fetcher that kept it would sign the reader
 * back in on the next page load. The refusal is not passed on either: callers clear the signed-in
 * state only once this resolves, and the credentials are gone by then.
 *
 * Only the backend can expire its HttpOnly cookie, so a request that failed, on the network or at
 * the rate limit, is sent a second time. The cookie script can reach is cleared in both its forms
 * whatever came of the request
 */
export function logout(): Promise<void> {
  const request = () => authFetcher.get<void>('/logout');

  return request()
    .catch((e) => {
      if (isRefused(e)) {
        return undefined;
      }
      return new Promise((resolve) => setTimeout(resolve, LOGOUT_RETRY_DELAY_MS)).then(request);
    })
    .catch(() => undefined)
    .finally(() => dropSession(true));
}
