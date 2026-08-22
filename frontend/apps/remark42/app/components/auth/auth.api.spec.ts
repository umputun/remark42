import { oauthSignin } from './auth.api';

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
