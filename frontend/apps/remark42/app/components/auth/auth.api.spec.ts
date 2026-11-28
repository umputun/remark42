import type { User } from 'common/types';
import { apiFetcher, JWT_HEADER } from 'common/fetcher';
import { user } from '__stubs__/user';
import { RequestError } from 'utils/errorUtils';

import * as cookies from 'common/cookies';

import {
  anonymousSignin,
  logout,
  oauthSignin,
  verifyEmailSignin,
  verifyTelegramSignin,
  LOGOUT_RETRY_DELAY_MS,
  HANDOFF_RETRY_DELAY_MS,
  HANDOFF_RETRIES,
  POPUP_BLOCKED,
} from './auth.api';
import * as handoff from './auth.handoff';

/**
 * The OAuth flow has no message from the popup to listen for: the popup closes itself and the
 * opener notices, so completion is detected by polling `getUser` whenever the document regains
 * focus. That polling is what these tests are about. Cross-domain embedding is the case where
 * the cookie never becomes readable and `getUser` therefore never stops returning null, so the
 * retry has to be bounded by something other than success.
 */
describe('oauthSignin', () => {
  // open when the click opens it, so it is not taken for a blocked one, and closed by the reader later
  let popupClosed = false;
  const openedWindow = {
    get closed() {
      return popupClosed;
    },
  } as unknown as Window;
  function signin(url: string) {
    const pending = oauthSignin(url);
    popupClosed = true;
    return pending;
  }

  beforeEach(() => {
    popupClosed = false;
    jest.useFakeTimers();
    jest.spyOn(window, 'open').mockReturnValue(openedWindow);
    // the poll only fires while the document is focused and visible
    jest.spyOn(document, 'hasFocus').mockReturnValue(true);
    Object.defineProperty(document, 'hidden', { value: false, configurable: true });
    fetchMock.resetMocks();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  /** how many listeners the flow has attached and not removed */
  function listenerCount() {
    return (
      (document.addEventListener as jest.Mock).mock.calls.length -
      (document.removeEventListener as jest.Mock).mock.calls.length
    );
  }

  it('stops polling once it gives up', async () => {
    // an unauthenticated response is what a cross-domain embed gets indefinitely
    fetchMock.mockResponse('', { status: 401 });
    const pending = signin('/auth/github/login');
    const rejected = expect(pending).rejects.toThrow(/authorization window/);

    // the poll is reachable only from the two listeners and from the retry it schedules itself,
    // so without dispatching one of them no request is ever made and this test asserts nothing
    window.dispatchEvent(new Event('focus'));
    await jest.advanceTimersByTimeAsync(0);
    expect(fetchMock.mock.calls.length).toBeGreaterThan(0);

    // it keeps retrying while getUser returns null
    await jest.advanceTimersByTimeAsync(60 * 1000);
    const whileRunning = fetchMock.mock.calls.length;
    expect(whileRunning).toBeGreaterThan(1);

    await jest.advanceTimersByTimeAsync(5 * 60 * 1000);
    await rejected;

    const afterGivingUp = fetchMock.mock.calls.length;
    // without the teardown the one-minute retry keeps rescheduling itself for the life of the
    // page, against a route capped at 2 req/s
    await jest.advanceTimersByTimeAsync(10 * 60 * 1000);
    expect(fetchMock.mock.calls.length).toBe(afterGivingUp);
  });

  it('does not reschedule when getUser resolves after the deadline', async () => {
    // the deadline can pass while a request is in flight. that resolution still runs the code
    // after the await, and without a guard it schedules a fresh retry with nothing left to clear
    let release: (v: Response) => void = () => undefined;
    fetchMock.mockImplementation(
      () =>
        new Promise<Response>((r) => {
          release = r as (v: Response) => void;
        })
    );

    const pending = signin('/auth/github/login');
    const rejected = expect(pending).rejects.toThrow(/authorization window/);

    window.dispatchEvent(new Event('focus'));
    await jest.advanceTimersByTimeAsync(0);
    expect(fetchMock.mock.calls.length).toBe(1);

    await jest.advanceTimersByTimeAsync(5 * 60 * 1000);
    await rejected;

    // the in-flight request now comes back unauthenticated, after the flow already gave up
    release(new Response('', { status: 401 }));
    await jest.advanceTimersByTimeAsync(0);

    const afterGivingUp = fetchMock.mock.calls.length;
    await jest.advanceTimersByTimeAsync(10 * 60 * 1000);
    expect(fetchMock.mock.calls.length).toBe(afterGivingUp);
  });

  it('detaches its listeners once it gives up', async () => {
    jest.spyOn(document, 'addEventListener');
    jest.spyOn(document, 'removeEventListener');
    fetchMock.mockResponse('', { status: 401 });

    const pending = signin('/auth/github/login');
    const rejected = expect(pending).rejects.toThrow();
    expect(listenerCount()).toBeGreaterThan(0);

    await jest.advanceTimersByTimeAsync(5 * 60 * 1000);
    await rejected;

    expect(listenerCount()).toBe(0);
  });

  it('rejects with an error rather than undefined', async () => {
    fetchMock.mockResponse('', { status: 401 });
    const pending = signin('/auth/github/login');
    // the caller stores whatever this rejects with as the error state, so rejecting with no
    // argument puts undefined on screen; the code is what the panel shows a message for
    const assertion = expect(pending).rejects.toMatchObject({
      code: 0,
      message: expect.stringMatching(/authorization window/),
    });

    await jest.advanceTimersByTimeAsync(5 * 60 * 1000);
    await assertion;
  });
});

/**
 * A sign-in endpoint answers with the auth library's user, which keeps the admin flag under
 * `attrs`. What reaches the store has to be the API's user instead, or an admin is shown as a
 * regular user until the page is reloaded (#2205)
 */
describe.each`
  name                      | signin
  ${'anonymousSignin'}      | ${() => anonymousSignin('username')}
  ${'verifyEmailSignin'}    | ${() => verifyEmailSignin('token')}
  ${'verifyTelegramSignin'} | ${() => verifyTelegramSignin('token')}
`('$name', ({ signin }: { signin: () => Promise<User> }) => {
  const json = { headers: { 'Content-Type': 'application/json' } };

  beforeEach(() => {
    fetchMock.resetMocks();
  });

  it('resolves the user the API knows rather than the one sign-in answered with', async () => {
    const admin = { ...user, admin: true };
    fetchMock.mockResponses(
      [JSON.stringify({ id: user.id, name: user.name, attrs: { admin: true } }), json],
      [JSON.stringify(admin), json]
    );

    await expect(signin()).resolves.toEqual(admin);
    expect(fetchMock.mock.calls).toHaveLength(2);
    expect(fetchMock.mock.calls[1][0]).toContain('/api/v1/user');
  });

  it('rejects when the signed-in user cannot be loaded', async () => {
    fetchMock.mockResponses(['{}', json], ['', { status: 401 }]);

    await expect(signin()).rejects.toBeInstanceOf(RequestError);
  });
});

describe('logout', () => {
  beforeEach(() => {
    fetchMock.resetMocks();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('forgets a header-held session even when the server refuses the logout', async () => {
    // a token delivered as a header is what the fetcher keeps in memory and sends back
    jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'jti' }));
    fetchMock.mockResponseOnce('{}', {
      headers: { [JWT_HEADER]: 'held.jwt.token', 'Content-Type': 'application/json' },
    });
    await apiFetcher.get('/anything');

    // an expired header token is refused with 403 by the logout handler
    fetchMock.mockResponseOnce(JSON.stringify({ error: 'logout not allowed' }), { status: 403 });
    // resolving lets the caller clear the signed-in state as well
    await expect(logout()).resolves.toBeUndefined();

    fetchMock.mockResponseOnce('{}', { headers: { 'Content-Type': 'application/json' } });
    await apiFetcher.get('/anything');
    const [, init] = fetchMock.mock.calls[fetchMock.mock.calls.length - 1];
    expect((init?.headers as Record<string, string>)[JWT_HEADER]).toBeUndefined();
  });

  /** the logout requests sent so far */
  function logoutRequests() {
    return fetchMock.mock.calls.filter(([url]) => String(url).includes('/auth/logout')).length;
  }

  it.each([
    ['never arrived', () => fetchMock.mockRejectOnce(new TypeError('Failed to fetch'))],
    ['was turned away by the rate limit', () => fetchMock.mockResponseOnce('', { status: 429 })],
  ])('sends the request once more when it %s', async (_, fail) => {
    jest.useFakeTimers();
    try {
      fail();
      fetchMock.mockResponseOnce('', { status: 200 });
      let resolved = false;
      const pending = logout().then(() => {
        resolved = true;
      });

      // the backend takes two requests a second on these routes, so the repeat waits
      await jest.advanceTimersByTimeAsync(LOGOUT_RETRY_DELAY_MS - 1);
      expect(logoutRequests()).toBe(1);
      expect(resolved).toBe(false);

      await jest.advanceTimersByTimeAsync(1);
      await pending;
      expect(logoutRequests()).toBe(2);
    } finally {
      jest.useRealTimers();
    }
  });

  it('gives up after the second failure, and still resolves', async () => {
    jest.useFakeTimers();
    try {
      const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
      fetchMock.mockReject(new TypeError('Failed to fetch'));
      const pending = logout();

      await jest.advanceTimersByTimeAsync(LOGOUT_RETRY_DELAY_MS);
      await expect(pending).resolves.toBeUndefined();
      await jest.advanceTimersByTimeAsync(10 * LOGOUT_RETRY_DELAY_MS);
      expect(logoutRequests()).toBe(2);
      // with no answer from the server, what script can clear is all that ends the session here
      expect(clear).toHaveBeenCalledWith('XSRF-TOKEN', true);
      expect(clear).toHaveBeenCalledWith('JWT', true);
    } finally {
      jest.useRealTimers();
    }
  });

  it.each([401, 403])('does not repeat a sign-out the server refused with %d', async (status) => {
    fetchMock.mockResponseOnce(JSON.stringify({ error: 'logout not allowed' }), { status });

    await logout();

    expect(logoutRequests()).toBe(1);
  });

  it('clears the cookies in both forms, whatever became of the request', async () => {
    const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
    fetchMock.mockResponseOnce(JSON.stringify({ error: 'logout not allowed' }), { status: 403 });

    await logout();

    // a frame on another host of the page's site holds the backend's XSRF cookie in both forms,
    // and the one left behind would go on signing this widget's requests
    expect(clear).toHaveBeenCalledWith('XSRF-TOKEN', true);
    expect(clear).toHaveBeenCalledWith('JWT', true);
  });
});

/**
 * The handoff: the popup opened on the start page tells this frame its transaction, and from then
 * on the transaction alone decides how the sign-in ends. The verifier is fixed here; making one
 * needs WebCrypto, which is not what these tests are about
 */
describe('oauthSignin with the handoff', () => {
  const txn = 'a'.repeat(32);
  const json = { 'Content-Type': 'application/json' };
  let popupClosed = false;
  const popup = {
    get closed() {
      return popupClosed;
    },
    postMessage: jest.fn(),
    close: jest.fn(),
  };
  const asWindow = popup as unknown as Window;

  beforeEach(() => {
    popupClosed = false;
    popup.postMessage.mockClear();
    popup.close.mockClear();
    jest.useFakeTimers();
    jest.spyOn(handoff, 'takeHandoff').mockReturnValue({ verifier: 'the-verifier', challenge: 'the-challenge' });
    jest.spyOn(window, 'open').mockReturnValue(asWindow);
    jest.spyOn(document, 'hasFocus').mockReturnValue(true);
    Object.defineProperty(document, 'hidden', { value: false, configurable: true });
    fetchMock.resetMocks();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  function postFromPopup(data: unknown, source: unknown = asWindow, origin = window.location.origin) {
    const evt = new MessageEvent('message', { data, origin });
    Object.defineProperty(evt, 'source', { value: source });
    window.dispatchEvent(evt);
  }

  /** answers each endpoint from the map, by the path the request goes to; anything else is 404 */
  function serve(routes: Record<string, () => { status?: number; body?: unknown }>) {
    fetchMock.mockResponse(async (req) => {
      const path = new URL(req.url).pathname;
      const route = Object.keys(routes).find((p) => path.endsWith(p));
      if (!route) {
        return { status: 404, body: '' };
      }
      const { status = 200, body = {} } = routes[route]();
      return { status, body: JSON.stringify(body), headers: json };
    });
  }

  function calls(path: string) {
    return fetchMock.mock.calls.filter(([u]) => new URL(String(u)).pathname.endsWith(path)).length;
  }

  it('opens the start page with the challenge, and answers the start page from its own popup only', async () => {
    serve({ '/handoff/status': () => ({ body: { status: 'pending' } }) });
    // left waiting, and given up on when the test's timers run out
    oauthSignin('https://remark42.example.com/auth/github/login?from=x&site=remark').catch(() => undefined);
    const opened = new URL((window.open as jest.Mock).mock.calls[0][0]);
    expect(opened.pathname).toBe('/auth/handoff/start');
    expect(opened.searchParams.get('provider')).toBe('github');
    expect(opened.searchParams.get('challenge')).toBe('the-challenge');
    expect(opened.searchParams.get('from')).toBe('x');
    expect(opened.searchParams.get('site')).toBe('remark');

    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn }, {});
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn }, asWindow, 'https://evil.example.com');
    postFromPopup({ type: 'other', txn });
    expect(popup.postMessage).not.toHaveBeenCalled();

    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    expect(popup.postMessage).toHaveBeenCalledWith({ type: handoff.HANDOFF_ACK, txn }, window.location.origin);
  });

  it('redeems the sign-in when the frame does not see the account', async () => {
    // a 401 on the way must not clear the cookies the sign-in has just set
    const clear = jest.spyOn(cookies, 'clearAuthCookie');
    let redeemed = false;
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/api/v1/user': () => (redeemed ? { body: user } : { status: 401 }),
      '/handoff/redeem': () => {
        redeemed = true;
        return { body: { user: user.id } };
      },
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });

    await expect(pending).resolves.toEqual(user);
    expect(calls('/handoff/redeem')).toBe(1);
    expect(popup.close).toHaveBeenCalledTimes(1);
    expect(clear).not.toHaveBeenCalled();
    const redeem = fetchMock.mock.calls.find(([u]) => String(u).includes('/handoff/redeem'));
    expect(JSON.parse(String(redeem?.[1]?.body))).toMatchObject({ txn, verifier: 'the-verifier' });
  });

  it('keeps the session the frame already sees when it is the account that signed in', async () => {
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/api/v1/user': () => ({ body: user }),
      '/handoff/cancel': () => ({ body: {} }),
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });

    await expect(pending).resolves.toEqual(user);
    expect(calls('/handoff/redeem')).toBe(0);
    expect(calls('/handoff/cancel')).toBe(1);
  });

  it('redeems when the frame sees another account, and rejects when the redeemed one is not it either', async () => {
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: 'someone-else' } }),
      '/api/v1/user': () => ({ body: user }),
      '/handoff/redeem': () => ({ body: {} }),
    });
    const pending = oauthSignin('/auth/github/login');
    const rejected = expect(pending).rejects.toBeInstanceOf(Error);
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });

    await rejected;
    expect(calls('/handoff/redeem')).toBe(1);
  });

  it('never takes an earlier session for the sign-in once the popup handed it over', async () => {
    serve({
      '/handoff/status': () => ({ body: { status: 'pending' } }),
      '/auth/status': () => ({ body: { status: 'logged in' } }),
      '/api/v1/user': () => ({ body: user }),
      '/handoff/cancel': () => ({ body: {} }),
    });
    const pending = oauthSignin('/auth/github/login');
    const rejected = expect(pending).rejects.toThrow(/authorization window/);
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(0);

    // the provider cut the popup off, so it reads as closed while the reader is still signing in
    popupClosed = true;
    window.dispatchEvent(new Event('focus'));
    await jest.advanceTimersByTimeAsync(5 * 60 * 1000);
    await rejected;

    expect(calls('/auth/status')).toBe(0);
    expect(calls('/api/v1/user')).toBe(0);
    await jest.advanceTimersByTimeAsync(0);
    expect(calls('/handoff/cancel')).toBe(1);
  });

  it('waits as before when the popup gave the handoff up', async () => {
    serve({
      '/handoff/status': () => ({ body: { status: 'canceled' } }),
      '/auth/status': () => ({ body: { status: 'logged in' } }),
      '/api/v1/user': () => ({ body: user }),
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(0);
    expect(calls('/handoff/status')).toBe(1);

    popupClosed = true;
    window.dispatchEvent(new Event('focus'));
    await expect(pending).resolves.toEqual(user);
    expect(calls('/handoff/redeem')).toBe(0);
  });

  it.each([
    ['429', 429],
    ['500', 500],
  ])('asks for the user again after a %s and does not redeem on it', async (_, failure) => {
    let first = true;
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/api/v1/user': () => {
        if (first) {
          first = false;
          return { status: failure };
        }
        return { body: user };
      },
      '/handoff/cancel': () => ({ body: {} }),
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(HANDOFF_RETRY_DELAY_MS);

    await expect(pending).resolves.toEqual(user);
    expect(calls('/handoff/redeem')).toBe(0);
  });

  it('takes the session it sees when the backend refuses the redeem for that reason', async () => {
    let userCalls = 0;
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      // the first answer is lost on the way, the backend then sees the frame's cookie for the account
      '/api/v1/user': () => (++userCalls === 1 ? { status: 401 } : { body: user }),
      '/handoff/redeem': () => ({ status: 409, body: { code: 3, error: 'signed in already' } }),
      '/handoff/cancel': () => ({ body: {} }),
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });

    await expect(pending).resolves.toEqual(user);
    expect(calls('/handoff/redeem')).toBe(1);
  });

  it('gives up without redeeming when the user cannot be loaded at all', async () => {
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/api/v1/user': () => ({ status: 500 }),
    });
    const pending = oauthSignin('/auth/github/login');
    const rejected = expect(pending).rejects.toEqual(expect.anything());
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(HANDOFF_RETRY_DELAY_MS * (HANDOFF_RETRIES + 1));

    await rejected;
    expect(calls('/handoff/redeem')).toBe(0);
  });

  it.each([
    ['a network failure as Firefox and Safari throw it', () => Promise.reject(new TypeError('Load failed'))],
    ['a gateway timeout', () => Promise.resolve(new Response('<html>504</html>', { status: 504 }))],
  ])('waits through %s on the status', async (_, failure) => {
    let first = true;
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/api/v1/user': () => ({ body: user }),
      '/handoff/cancel': () => ({ status: 204 }),
    });
    const served = fetchMock.getMockImplementation()!;
    fetchMock.mockImplementation((req, init) => {
      const u = String(req instanceof Request ? req.url : req);
      if (first && u.endsWith('/handoff/status')) {
        first = false;
        return failure() as Promise<Response>;
      }
      return served(req, init);
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(HANDOFF_RETRY_DELAY_MS);

    await expect(pending).resolves.toEqual(user);
  });

  it("redeems when the frame's session is for another site served from this host", async () => {
    let redeemed = false;
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      // the backend's own code for a refusal, which the fetcher keeps as the code, with 403 as the status
      '/api/v1/user': () => (redeemed ? { body: user } : { status: 403, body: { code: 3, error: 'site mismatch' } }),
      '/handoff/redeem': () => {
        redeemed = true;
        return { status: 204 };
      },
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });

    await expect(pending).resolves.toEqual(user);
    expect(calls('/handoff/redeem')).toBe(1);
  });

  it('leaves the redeem to a newer attempt started while it looked the account up', async () => {
    let releaseUser: () => void = () => undefined;
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/handoff/redeem': () => ({ status: 204 }),
      '/handoff/cancel': () => ({ status: 204 }),
    });
    const served = fetchMock.getMockImplementation()!;
    fetchMock.mockImplementation((req, init) => {
      const u = String(req instanceof Request ? req.url : req);
      if (u.includes('/api/v1/user')) {
        return new Promise<Response>((resolve) => {
          releaseUser = () => resolve(new Response('', { status: 401 }));
        });
      }
      return served(req, init);
    });
    const pending = oauthSignin('/auth/github/login');
    pending.catch(() => undefined);
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(0);

    // the reader clicks again while the first attempt waits for /user
    oauthSignin('/auth/github/login');
    releaseUser();
    await jest.advanceTimersByTimeAsync(0);
    expect(calls('/handoff/redeem')).toBe(0);
  });

  it('asks for the redeemed account again after a request that did not get through', async () => {
    let redeemed = false;
    let afterRedeem = 0;
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/api/v1/user': () => {
        if (!redeemed) {
          return { status: 401 };
        }
        return ++afterRedeem === 1 ? { status: 429 } : { body: user };
      },
      '/handoff/redeem': () => {
        redeemed = true;
        return { status: 204 };
      },
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(HANDOFF_RETRY_DELAY_MS);

    await expect(pending).resolves.toEqual(user);
  });

  it('redeems again after the rate limit refused it, and shows no debug text on a failure', async () => {
    let redeems = 0;
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/api/v1/user': () => (redeems >= 2 ? { body: user } : { status: 401 }),
      '/handoff/redeem': () => (++redeems === 1 ? { status: 429 } : { status: 204 }),
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(HANDOFF_RETRY_DELAY_MS);
    await expect(pending).resolves.toEqual(user);
    expect(redeems).toBe(2);
  });

  it('does not redeem again for an attempt a new click replaced while it waited out the rate limit', async () => {
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/api/v1/user': () => ({ status: 401 }),
      '/handoff/redeem': () => ({ status: 429 }),
      '/handoff/cancel': () => ({ status: 204 }),
    });
    const pending = oauthSignin('/auth/github/login');
    pending.catch(() => undefined);
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(0);
    expect(calls('/handoff/redeem')).toBe(1);

    oauthSignin('/auth/github/login');
    await jest.advanceTimersByTimeAsync(HANDOFF_RETRY_DELAY_MS);
    expect(calls('/handoff/redeem')).toBe(1);
  });

  it('reports a request that never got an answer as the fetcher does', async () => {
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: user.id } }),
      '/api/v1/user': () => ({ status: 401 }),
    });
    const served = fetchMock.getMockImplementation()!;
    fetchMock.mockImplementation((req, init) => {
      const u = String(req instanceof Request ? req.url : req);
      if (u.endsWith('/handoff/redeem')) {
        return Promise.reject(new TypeError('Load failed'));
      }
      return served(req, init);
    });
    const pending = oauthSignin('/auth/github/login');
    const rejected = expect(pending).rejects.toMatchObject({ code: 'fetch-error' });
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await rejected;
  });

  it('looks at a popup that closed while its cancelled handoff was still reported', async () => {
    let releaseStatus: () => void = () => undefined;
    serve({
      '/auth/status': () => ({ body: { status: 'logged in' } }),
      '/api/v1/user': () => ({ body: user }),
    });
    const served = fetchMock.getMockImplementation()!;
    fetchMock.mockImplementation((req, init) => {
      const u = String(req instanceof Request ? req.url : req);
      if (u.endsWith('/handoff/status')) {
        return new Promise<Response>((resolve) => {
          releaseStatus = () =>
            resolve(
              new Response(JSON.stringify({ status: 'canceled' }), { headers: { 'Content-Type': 'application/json' } })
            );
        });
      }
      return served(req, init);
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(0);

    // the popup closes and the frame regains focus while the cancel is not reported yet
    popupClosed = true;
    window.dispatchEvent(new Event('focus'));
    await jest.advanceTimersByTimeAsync(0);
    releaseStatus();

    await expect(pending).resolves.toEqual(user);
  });

  it("reports a proxy's page that is not JSON with a code the panel has a message for", async () => {
    serve({});
    const served = fetchMock.getMockImplementation()!;
    fetchMock.mockImplementation((req, init) => {
      const u = String(req instanceof Request ? req.url : req);
      if (u.endsWith('/handoff/status')) {
        return Promise.resolve(new Response('<html>maintenance</html>', { status: 200 }));
      }
      return served(req, init);
    });
    const pending = oauthSignin('/auth/github/login');
    const rejected = expect(pending).rejects.toMatchObject({ code: 0 });
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await rejected;
  });

  it('reports a failed handoff with the general message', async () => {
    serve({ '/handoff/status': () => ({ status: 404, body: {} }) });
    const pending = oauthSignin('/auth/github/login');
    const rejected = expect(pending).rejects.toMatchObject({
      error: 'Something went wrong. Please try again a bit later.',
    });
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await rejected;
  });

  it('closes the popup only once the frame is signed in', async () => {
    serve({
      '/handoff/status': () => ({ body: { status: 'done', user: 'someone-else' } }),
      '/api/v1/user': () => ({ body: user }),
      '/handoff/redeem': () => ({ status: 204 }),
    });
    const pending = oauthSignin('/auth/github/login');
    const rejected = expect(pending).rejects.toEqual(expect.anything());
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await rejected;
    expect(popup.close).not.toHaveBeenCalled();
  });

  it('asks again after a request that did not get through', async () => {
    let first = true;
    serve({
      '/handoff/status': () => {
        if (first) {
          first = false;
          return { status: 429 };
        }
        return { body: { status: 'done', user: user.id } };
      },
      '/api/v1/user': () => ({ body: user }),
      '/handoff/cancel': () => ({ body: {} }),
    });
    const pending = oauthSignin('/auth/github/login');
    postFromPopup({ type: handoff.HANDOFF_MESSAGE, txn });
    await jest.advanceTimersByTimeAsync(HANDOFF_RETRY_DELAY_MS);

    await expect(pending).resolves.toEqual(user);
    expect(calls('/handoff/status')).toBe(2);
  });

  it.each([
    ['null', null],
    ['a window closed from the start', { closed: true }],
  ])('rejects at once when the browser blocks the window: %s', async (_, opened) => {
    (window.open as jest.Mock).mockReturnValue(opened);
    await expect(oauthSignin('/auth/github/login')).rejects.toMatchObject({ code: POPUP_BLOCKED });
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
