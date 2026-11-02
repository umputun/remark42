import type { User } from 'common/types';
import { apiFetcher, JWT_HEADER } from 'common/fetcher';
import { user } from '__stubs__/user';
import { RequestError } from 'utils/errorUtils';

import { anonymousSignin, logout, oauthSignin, verifyEmailSignin, verifyTelegramSignin } from './auth.api';

/**
 * The OAuth flow has no message from the popup to listen for: the popup closes itself and the
 * opener notices, so completion is detected by polling `getUser` whenever the document regains
 * focus. That polling is what these tests are about. Cross-domain embedding is the case where
 * the cookie never becomes readable and `getUser` therefore never stops returning null, so the
 * retry has to be bounded by something other than success.
 */
describe('oauthSignin', () => {
  const openedWindow = { closed: true } as Window;

  beforeEach(() => {
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
    const pending = oauthSignin('/auth/github/login');
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

    const pending = oauthSignin('/auth/github/login');
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

    const pending = oauthSignin('/auth/github/login');
    const rejected = expect(pending).rejects.toThrow();
    expect(listenerCount()).toBeGreaterThan(0);

    await jest.advanceTimersByTimeAsync(5 * 60 * 1000);
    await rejected;

    expect(listenerCount()).toBe(0);
  });

  it('rejects with an error rather than undefined', async () => {
    fetchMock.mockResponse('', { status: 401 });
    const pending = oauthSignin('/auth/github/login');
    // the caller stores whatever this rejects with as the error state, so rejecting with no
    // argument puts undefined on screen
    const assertion = expect(pending).rejects.toBeInstanceOf(Error);

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
});
