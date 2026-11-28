import { waitFor } from '@testing-library/preact';

import { mockStore } from '__stubs__/store';
import * as api from 'common/api';
import * as fetcher from 'common/fetcher';
import * as authApi from 'components/auth/auth.api';
import type { User } from 'common/types';
import { COMMENTS_SET } from 'store/comments/types';

import INITIAL_STORE from './__stubs__/comments-store.json';

import { followReaderAcrossFrames, signin, signout } from './actions';
import { USER_SET } from './types';

/** a channel that records what this frame posts, and lets a test post as another frame */
class FakeChannel {
  static last: FakeChannel | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  posted: unknown[] = [];

  constructor(readonly name: string) {
    FakeChannel.last = this;
  }

  postMessage(message: unknown) {
    this.posted.push(message);
  }

  close() {
    // nothing to release
  }

  /** a message from another frame, which the channel never delivers to the one that posted it */
  receive(data: 'signin' | 'signout' = 'signin') {
    this.onmessage?.({ data } as MessageEvent);
  }
}

const info = { url: 'https://example.com/post', count: 0 };

describe('the reader across widget frames', () => {
  const { BroadcastChannel } = window;

  beforeEach(() => {
    window.BroadcastChannel = FakeChannel as unknown as typeof window.BroadcastChannel;
    jest.spyOn(api, 'getPostComments').mockResolvedValue({ comments: [], info });
  });

  afterEach(() => {
    window.BroadcastChannel = BroadcastChannel;
    jest.restoreAllMocks();
  });

  it('shows the reader who signed in in another frame and loads the comments for them', async () => {
    const store = mockStore({ ...INITIAL_STORE, user: null, hiddenUsers: {} });
    jest.spyOn(api, 'getUser').mockResolvedValue({ id: '1' } as User);

    store.dispatch(followReaderAcrossFrames());
    FakeChannel.last!.receive();

    await waitFor(() => expect(store.getActions()).toContainEqual({ type: COMMENTS_SET, comments: [] }));
    expect(store.getActions()).toContainEqual({ type: USER_SET, user: { id: '1' } });
  });

  it('goes on without the notices where the browser refuses the frame a channel', async () => {
    const store = mockStore({ ...INITIAL_STORE, user: null, hiddenUsers: {} });
    window.BroadcastChannel = class {
      constructor() {
        throw new DOMException('The operation is insecure.', 'SecurityError');
      }
    } as unknown as typeof window.BroadcastChannel;

    expect(() => store.dispatch(followReaderAcrossFrames())).not.toThrow();
  });

  it('loads nothing again when the reader it shows is the one signed in', async () => {
    const store = mockStore({ ...INITIAL_STORE, user: { id: '1' }, hiddenUsers: {} });
    const getUser = jest.spyOn(api, 'getUser').mockResolvedValue({ id: '1' } as User);

    store.dispatch(followReaderAcrossFrames());
    FakeChannel.last!.receive();

    await waitFor(() => expect(getUser).toHaveBeenCalled());
    await Promise.resolve();
    expect(api.getPostComments).not.toHaveBeenCalled();
  });

  it('tells the other frames when the reader signs in here', async () => {
    const store = mockStore({ ...INITIAL_STORE, user: null, hiddenUsers: {} });

    store.dispatch(followReaderAcrossFrames());
    await store.dispatch(signin({ id: '1' } as User));

    expect(FakeChannel.last!.name).toBe('remark42-reader');
    expect(FakeChannel.last!.posted).toEqual(['signin']);
  });

  it('tells the other frames when the reader signs out here', async () => {
    const store = mockStore({ ...INITIAL_STORE, user: { id: '1' }, hiddenUsers: {} });
    jest.spyOn(authApi, 'logout').mockResolvedValue(undefined);

    store.dispatch(followReaderAcrossFrames());
    await store.dispatch(signout());

    expect(FakeChannel.last!.posted).toEqual(['signout']);
  });

  // with the token sent in a header, a frame holds a copy of it that the cookies going away do not end
  it('forgets the session it holds when the reader signs out in another frame', async () => {
    const store = mockStore({ ...INITIAL_STORE, user: { id: '1' }, hiddenUsers: {} });
    const dropSession = jest.spyOn(fetcher, 'dropSession');
    jest.spyOn(api, 'getUser').mockResolvedValue(null);

    store.dispatch(followReaderAcrossFrames());
    FakeChannel.last!.receive('signout');

    expect(dropSession).toHaveBeenCalled();
    await waitFor(() => expect(store.getActions()).toContainEqual({ type: USER_SET, user: null }));
  });

  it('keeps a sign-in in another frame from dropping the session it holds', async () => {
    const store = mockStore({ ...INITIAL_STORE, user: null, hiddenUsers: {} });
    const dropSession = jest.spyOn(fetcher, 'dropSession');
    const getUser = jest.spyOn(api, 'getUser').mockResolvedValue({ id: '1' } as User);

    store.dispatch(followReaderAcrossFrames());
    FakeChannel.last!.receive('signin');

    await waitFor(() => expect(getUser).toHaveBeenCalled());
    expect(dropSession).not.toHaveBeenCalled();
  });

  it('shows the reader of the latest notice when an earlier answer comes back last', async () => {
    const store = mockStore({ ...INITIAL_STORE, user: null, hiddenUsers: {} });
    let answerFirst: (user: User | null) => void = () => undefined;
    jest
      .spyOn(api, 'getUser')
      .mockReturnValueOnce(new Promise((resolve) => (answerFirst = resolve)))
      .mockResolvedValueOnce(null);

    store.dispatch(followReaderAcrossFrames());
    FakeChannel.last!.receive('signin');
    FakeChannel.last!.receive('signout');
    await waitFor(() => expect(store.getActions()).toContainEqual({ type: USER_SET, user: null }));
    answerFirst({ id: '1' } as User);
    await Promise.resolve();
    await Promise.resolve();

    expect(store.getActions()).not.toContainEqual({ type: USER_SET, user: { id: '1' } });
  });
});
