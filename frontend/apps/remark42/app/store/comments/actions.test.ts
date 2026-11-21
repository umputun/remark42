import { mockStore } from '__stubs__/store';
import { LS_SORT_KEY } from 'common/constants';

import * as api from 'common/api';
import { POST_INFO_SET } from 'store/post-info/types';

import { fetchComments, reloadComments, updateSorting } from './actions';
import { COMMENTS_REQUEST_FETCHING, COMMENTS_REQUEST_SUCCESS, COMMENTS_SET, COMMENTS_SET_SORT } from './types';

describe('Store comments actions', () => {
  it('should save last sort to localstorage', async () => {
    const newSort = '+controversy';
    const store = mockStore({
      comments: {
        sort: '+active',
      },
      hiddenUsers: {},
    });

    await store.dispatch(updateSorting(newSort));

    const [setCommentAction] = store.getActions();

    expect(setCommentAction).toEqual({ type: COMMENTS_SET_SORT, payload: newSort });
    expect(localStorage.setItem).toHaveBeenCalledWith(LS_SORT_KEY, newSort);
  });

  describe('loading the thread', () => {
    const info = { url: 'https://example.com/post', count: 0 };
    const state = { comments: { sort: '+active' }, hiddenUsers: {} };

    beforeEach(() => {
      jest.spyOn(api, 'getPostComments').mockResolvedValueOnce({ comments: [], info });
    });

    it('marks the request as running when fetching, which shows the preloader', async () => {
      const store = mockStore(state);

      await store.dispatch(fetchComments());

      expect(store.getActions()).toEqual([
        { type: COMMENTS_REQUEST_FETCHING },
        { type: COMMENTS_REQUEST_SUCCESS },
        { type: COMMENTS_SET, comments: [] },
        { type: POST_INFO_SET, info },
      ]);
    });

    it('leaves the loaded comments on screen when reloading', async () => {
      const store = mockStore(state);

      await store.dispatch(reloadComments());

      expect(store.getActions()).toEqual([
        { type: COMMENTS_SET, comments: [] },
        { type: POST_INFO_SET, info },
      ]);
    });
  });
});
