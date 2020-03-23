import { mockStore } from '@app/testUtils/mockStore';
import { User } from '@app/common/types';
import { LS_HIDDEN_USERS_KEY } from '@app/common/constants';
import { COMMENTS_PATCH } from '@app/store/comments/types';

import INITIAL_STORE from './__mocks__/comments-store.json';
import { setVerifiedStatus, hideUser, unhideUser, unblockUser } from './actions';
import { USER_UNHIDE, USER_HIDE, USER_UNBAN } from './types';

describe('store user actions', () => {
  beforeAll(() => {
    require('jest-fetch-mock').enableMocks();
  });
  afterAll(() => {
    require('jest-fetch-mock').resetMocks();
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

  describe('hide/unhide comments of user', () => {
    beforeEach(() => {
      localStorage.clear();
    });

    const store = mockStore(INITIAL_STORE);
    test('hideUser', async () => {
      await store.dispatch(hideUser({ id: '1' } as User));

      const actions = store.getActions();

      expect(actions[0]).toEqual({ type: USER_HIDE, user: { id: '1' } });
      expect(actions[1]).toEqual({ type: COMMENTS_PATCH, ids: ['1', '3'], patch: { hidden: true } });
      expect(localStorage.getItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY);
      expect(localStorage.setItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY, JSON.stringify({ '1': { id: '1' } }));
    });

    test('unhideUser', async () => {
      localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify({ '1': { id: '1' } }));
      await store.dispatch(unhideUser('1'));

      const actions = store.getActions();

      expect(actions[2]).toEqual({ type: USER_UNHIDE, id: '1' });
      expect(localStorage.getItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY);
      expect(localStorage.setItem).toHaveBeenCalledWith(LS_HIDDEN_USERS_KEY, JSON.stringify({}));
    });
  });
});
