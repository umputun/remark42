import stubStore from '__stubs__/store';
import { User } from 'common/types';
import { LS_HIDDEN_USERS_KEY } from 'common/constants';
import { COMMENTS_PATCH } from 'store/comments/types';

import INITIAL_STORE from './__stubs__/comments-store.json';
import { setVerifiedStatus, hideUser, unhideUser, unblockUser, fetchBlockedUsers, fetchHiddenUsers } from './actions';
import { USER_UNHIDE, USER_HIDE, USER_UNBAN, USER_BANLIST_SET, USER_HIDELIST_SET } from './types';

describe('store user actions', () => {
  test('fetchBlockedUsers', async () => {
    const store = stubStore(INITIAL_STORE);

    await store.dispatch(fetchBlockedUsers());

    const actions = store.getActions();

    expect(actions[0]).toEqual({ type: USER_BANLIST_SET, list: [] });
  });

  test('fetchHiddenUsers', async () => {
    const store = stubStore(INITIAL_STORE);

    await store.dispatch(fetchHiddenUsers());

    const actions = store.getActions();

    expect(actions[0]).toEqual({ type: USER_HIDELIST_SET, payload: {} });
  });

  test('fetchHiddenUsers with data', async () => {
    const data = { '1': { id: '1' }, '2': { id: '2' } };
    const store = stubStore(INITIAL_STORE);

    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(data));
    await store.dispatch(fetchHiddenUsers());

    const actions = store.getActions();

    expect(actions[0]).toEqual({ type: USER_HIDELIST_SET, payload: data });
  });

  test('setVerifiedStatus', async () => {
    const store = stubStore(INITIAL_STORE);

    await store.dispatch(setVerifiedStatus('1', true));

    const actions = store.getActions();

    expect(actions[0]).toEqual({
      type: COMMENTS_PATCH,
      ids: ['1', '3'],
      patch: { user: { id: '1', verified: true } },
    });
  });

  test('unblockUser', async () => {
    const store = stubStore(INITIAL_STORE);

    await store.dispatch(unblockUser('1'));

    const actions = store.getActions();

    expect(actions[0]).toEqual({ type: USER_UNBAN, id: '1' });
    expect(actions[1]).toEqual({ type: COMMENTS_PATCH, ids: ['1', '3'], patch: { user: { id: '1', block: false } } });
  });

  describe('hide/unhide comments of user', () => {
    beforeEach(() => {
      localStorage.clear();
    });

    test('hideUser', async () => {
      const store = stubStore(INITIAL_STORE);

      await store.dispatch(hideUser({ id: '1' } as User));

      const actions = store.getActions();

      expect(actions[0]).toEqual({ type: USER_HIDE, user: { id: '1' } });
      expect(actions[1]).toEqual({ type: COMMENTS_PATCH, ids: ['1', '3'], patch: { hidden: true } });
      expect(localStorage.getItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY);
      expect(localStorage.setItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY, JSON.stringify({ '1': { id: '1' } }));
    });

    test('unhideUser', async () => {
      const store = stubStore(INITIAL_STORE);

      localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify({ '1': { id: '1' } }));
      await store.dispatch(unhideUser('1'));

      const actions = store.getActions();

      expect(actions[0]).toEqual({ type: USER_UNHIDE, id: '1' });
      expect(localStorage.getItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY);
      expect(localStorage.setItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY, JSON.stringify({}));
    });
  });
});
