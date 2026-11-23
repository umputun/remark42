import { waitFor } from '@testing-library/preact';

import { mockStore } from '__stubs__/store';
import * as api from 'common/api';
import type { User } from 'common/types';
import { LS_HIDDEN_USERS_KEY } from 'common/constants';
import { COMMENTS_PATCH, COMMENTS_SET } from 'store/comments/types';
import { POST_INFO_SET } from 'store/post-info/types';
import { apiFetcher, JWT_HEADER } from 'common/fetcher';

import INITIAL_STORE from './__stubs__/comments-store.json';
import {
  setVerifiedStatus,
  hideUser,
  unhideUser,
  unblockUser,
  fetchBlockedUsers,
  fetchHiddenUsers,
  signin,
  signout,
} from './actions';
import { USER_UNHIDE, USER_HIDE, USER_UNBAN, USER_BANLIST_SET, USER_HIDELIST_SET, USER_SET } from './types';

describe('store user actions', () => {
  test('fetchBlockedUsers', async () => {
    const store = mockStore(INITIAL_STORE);

    await store.dispatch(fetchBlockedUsers());

    const actions = store.getActions();

    expect(actions[0]).toEqual({ type: USER_BANLIST_SET, list: [] });
  });

  test('fetchHiddenUsers', async () => {
    const store = mockStore(INITIAL_STORE);

    await store.dispatch(fetchHiddenUsers());

    const actions = store.getActions();

    expect(actions[0]).toEqual({ type: USER_HIDELIST_SET, payload: {} });
  });

  test('fetchHiddenUsers with data', async () => {
    const data = { '1': { id: '1' }, '2': { id: '2' } };
    const store = mockStore(INITIAL_STORE);

    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(data));
    await store.dispatch(fetchHiddenUsers());

    const actions = store.getActions();

    expect(actions[0]).toEqual({ type: USER_HIDELIST_SET, payload: data });
  });

  test('setVerifiedStatus', async () => {
    const store = mockStore(INITIAL_STORE);

    await store.dispatch(setVerifiedStatus('1', true));

    const actions = store.getActions();

    expect(actions[0]).toEqual({
      type: COMMENTS_PATCH,
      ids: ['1', '3'],
      patch: { user: { id: '1', verified: true } },
    });
  });

  test('unblockUser', async () => {
    const store = mockStore(INITIAL_STORE);

    await store.dispatch(unblockUser('1'));

    const actions = store.getActions();

    expect(actions[0]).toEqual({ type: USER_UNBAN, id: '1' });
    expect(actions[1]).toEqual({ type: COMMENTS_PATCH, ids: ['1', '3'], patch: { user: { id: '1', block: false } } });
  });

  test('unblockUser keeps the author on the comments of a user who did not write the first comment', async () => {
    const store = mockStore(INITIAL_STORE);

    await store.dispatch(unblockUser('2'));

    const actions = store.getActions();

    expect(actions[0]).toEqual({ type: USER_UNBAN, id: '2' });
    expect(actions[1]).toEqual({ type: COMMENTS_PATCH, ids: ['2'], patch: { user: { id: '2', block: false } } });
  });

  describe('hide/unhide comments of user', () => {
    beforeEach(() => {
      localStorage.clear();
    });

    test('hideUser', async () => {
      const store = mockStore(INITIAL_STORE);

      await store.dispatch(hideUser({ id: '1' } as User));

      const actions = store.getActions();

      expect(actions[0]).toEqual({ type: USER_HIDE, user: { id: '1' } });
      expect(actions[1]).toEqual({ type: COMMENTS_PATCH, ids: ['1', '3'], patch: { hidden: true } });
      expect(localStorage.getItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY);
      expect(localStorage.setItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY, JSON.stringify({ '1': { id: '1' } }));
    });

    test('unhideUser', async () => {
      const store = mockStore(INITIAL_STORE);

      localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify({ '1': { id: '1' } }));
      await store.dispatch(unhideUser('1'));

      const actions = store.getActions();

      expect(actions[0]).toEqual({ type: USER_UNHIDE, id: '1' });
      expect(localStorage.getItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY);
      expect(localStorage.setItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY, JSON.stringify({}));
    });
  });

  test('signout clears the user when the server refuses the logout', async () => {
    // signout reloads the comments, which reads the hidden users
    const store = mockStore({ ...INITIAL_STORE, hiddenUsers: {} });
    // an expired header token is refused with 403, and the reader still has to end up signed out
    fetchMock.mockResponseOnce(JSON.stringify({ error: 'logout not allowed' }), { status: 403 });

    await store.dispatch(signout());

    expect(store.getActions()).toContainEqual({ type: USER_SET, user: null });
  });

  test('signin reloads the comments behind the ones on screen', async () => {
    // marking the request as running puts a preloader in place of the list, which shrinks the
    // widget for the length of the request and loses the reader's place in a long thread
    const store = mockStore({ ...INITIAL_STORE, hiddenUsers: {} });
    const info = { url: 'https://example.com/post', count: 0 };
    jest.spyOn(api, 'getPostComments').mockResolvedValueOnce({ comments: [], info });

    await store.dispatch(signin({ id: '1' } as User));

    await waitFor(() => expect(store.getActions()).toContainEqual({ type: POST_INFO_SET, info }));
    expect(store.getActions()).toEqual([
      { type: USER_SET, user: { id: '1' } },
      { type: COMMENTS_SET, comments: [] },
      { type: POST_INFO_SET, info },
    ]);
  });

  test('signin keeps the reader signed in when the comments fail to reload', async () => {
    const store = mockStore({ ...INITIAL_STORE, hiddenUsers: {} });
    const failure = new Error('no answer');
    jest.spyOn(api, 'getPostComments').mockRejectedValueOnce(failure);
    const logged = jest.spyOn(console, 'error').mockImplementationOnce(() => undefined);

    await store.dispatch(signin({ id: '1' } as User));

    // the reload is not awaited by signin, so its failure has to be handled there and not left
    // as an unhandled rejection
    await waitFor(() => expect(logged).toHaveBeenCalledWith('Remark42: failed to reload comments', failure));
    // read once the failure has been handled, so a handler that cleared the user would show
    expect(store.getActions()).toEqual([{ type: USER_SET, user: { id: '1' } }]);
  });

  test('signout shows the reader as signed out and reloads the comments only once the logout is answered', async () => {
    const store = mockStore({ ...INITIAL_STORE, hiddenUsers: {} });
    fetchMock.resetMocks();
    // a token delivered as a header is what the fetcher keeps in memory and sends back
    jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'jti' }));
    fetchMock.mockResponseOnce('{}', {
      headers: { [JWT_HEADER]: 'held.jwt.token', 'Content-Type': 'application/json' },
    });
    await apiFetcher.get('/anything');
    fetchMock.mockClear();

    let answerLogout: (response: Response) => void = () => undefined;
    fetchMock.mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          answerLogout = resolve;
        })
    );
    fetchMock.mockResponse('{}', { headers: { 'Content-Type': 'application/json' } });

    const done = store.dispatch(signout());
    const requests = () =>
      fetchMock.mock.calls.map(([url, init]) => ({
        url: String(url),
        jwt: (init?.headers as Record<string, string>)[JWT_HEADER],
      }));

    // this ordering is what a sign-out asked for in the profile relies on as well, now that it
    // runs this action: comments reloaded ahead of the logout are answered with a renewed session
    await waitFor(() => expect(requests()).toHaveLength(1));
    expect(requests()[0].url).toContain('/auth/logout');
    expect(requests()[0].jwt).toBe('held.jwt.token');
    expect(store.getActions()).toEqual([]);

    answerLogout(new Response('', { status: 200 }));
    await done;

    expect(store.getActions()).toContainEqual({ type: USER_SET, user: null });
    // and nothing after the logout is signed as the reader
    expect(requests().length).toBeGreaterThan(1);
    expect(
      requests()
        .slice(1)
        .map((r) => r.jwt)
    ).toEqual(
      requests()
        .slice(1)
        .map(() => undefined)
    );
    expect(
      requests()
        .slice(1)
        .some((r) => r.url.includes('/auth/logout'))
    ).toBe(false);
  });
});
