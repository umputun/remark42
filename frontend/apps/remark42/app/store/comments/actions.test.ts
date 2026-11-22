import { applyMiddleware, combineReducers, createStore } from 'redux';
import thunk from 'redux-thunk';
import { waitFor } from '@testing-library/preact';

import { mockStore } from '__stubs__/store';
import { LS_SORT_KEY } from 'common/constants';

import * as api from 'common/api';
import type { Comment, Node, Tree, User } from 'common/types';
import type { StoreDispatch } from 'store';
import { rootProvider } from 'store/reducers';
import { POST_INFO_SET } from 'store/post-info/types';
import { hideUser } from 'store/user/actions';

import {
  addComment,
  fetchComments,
  patchComment,
  reloadComments,
  setComments,
  updateComment,
  updateSorting,
} from './actions';
import { COMMENTS_REQUEST_FETCHING, COMMENTS_REQUEST_SUCCESS, COMMENTS_SET, COMMENTS_SET_SORT } from './types';

const author: User = { id: 'author', name: 'author', picture: '', admin: false, ip: '', block: false, verified: false };

function comment(id: string, patch: Partial<Comment> = {}): Comment {
  return {
    id,
    pid: '',
    text: '<p>text</p>',
    user: author,
    locator: { site: 'remark', url: 'https://example.com/post' },
    score: 0,
    vote: 0,
    controversy: 0,
    time: '2026-10-31T12:00:00Z',
    title: '',
    ...patch,
  } as Comment;
}

const node = (id: string, patch: Partial<Comment> = {}): Node => ({ comment: comment(id, patch) });

/** a request whose response arrives when the test says so */
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}

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

  // the widget's own reducers: the mock store never changes its state, and these tests are about
  // what a load does when the store changed while it was waiting
  describe('a load the reader got ahead of', () => {
    const info = { url: 'https://example.com/post', count: 1 };
    const tree = (...comments: Node[]): Tree => ({ comments, info }) as Tree;

    function setup() {
      const store = createStore(combineReducers(rootProvider), applyMiddleware(thunk));
      const dispatch: StoreDispatch = store.dispatch;
      const getPostComments = jest.spyOn(api, 'getPostComments').mockReset();

      dispatch(setComments([node('c1')]));

      /** queues a response for the next request and hands back its release */
      const nextResponse = () => {
        const response = deferred<Tree>();
        getPostComments.mockReturnValueOnce(response.promise);
        return response;
      };

      return { dispatch, getPostComments, nextResponse, comments: () => store.getState().comments };
    }

    it('takes the response when nothing changed while it was on its way', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree(node('c1', { vote: 1, score: 1 })));

      await dispatch(reloadComments());

      expect(getPostComments).toHaveBeenCalledTimes(1);
      expect(comments().allComments.c1.vote).toBe(1);
    });

    it.each([
      {
        change: 'posted a comment',
        make: async (dispatch: StoreDispatch) => {
          jest.spyOn(api, 'addComment').mockResolvedValueOnce(comment('c2'));
          await dispatch(addComment('text', 'title'));
        },
        current: tree(node('c2'), node('c1')),
        kept: (comments: ReturnType<ReturnType<typeof setup>['comments']>) =>
          expect(comments.topComments).toEqual(['c2', 'c1']),
      },
      {
        change: 'edited a comment',
        make: async (dispatch: StoreDispatch) => {
          jest.spyOn(api, 'updateComment').mockResolvedValueOnce(comment('c1', { text: '<p>edited</p>' }));
          await dispatch(updateComment('c1', 'edited'));
        },
        current: tree(node('c1', { text: '<p>edited</p>' })),
        kept: (comments: ReturnType<ReturnType<typeof setup>['comments']>) =>
          expect(comments.allComments.c1.text).toBe('<p>edited</p>'),
      },
      {
        change: 'voted',
        make: async (dispatch: StoreDispatch) => {
          dispatch(patchComment({ id: 'c1', score: 1, vote: 1 }));
        },
        current: tree(node('c1', { vote: 1, score: 1 })),
        kept: (comments: ReturnType<ReturnType<typeof setup>['comments']>) =>
          expect(comments.allComments.c1.vote).toBe(1),
      },
    ])('leaves out the response and requests again when the reader $change meanwhile', async (c) => {
      const { dispatch, getPostComments, nextResponse, comments } = setup();
      const older = nextResponse();
      const newer = nextResponse();

      const reload = dispatch(reloadComments());
      await c.make(dispatch);
      // how the server had the thread before the reader's change
      older.resolve(tree(node('c1')));

      await waitFor(() => expect(getPostComments).toHaveBeenCalledTimes(2));
      // the older response has arrived by now, and the reader's change is still in the store
      c.kept(comments());

      newer.resolve(c.current);
      await reload;

      c.kept(comments());
      expect(getPostComments).toHaveBeenCalledTimes(2);
    });

    it('requests again with the sort chosen meanwhile', async () => {
      const { dispatch, getPostComments, nextResponse } = setup();
      const older = nextResponse();
      const newer = nextResponse();

      const reload = dispatch(reloadComments());
      dispatch({ type: COMMENTS_SET_SORT, payload: '+controversy' });
      dispatch(patchComment({ id: 'c1', score: 1, vote: 1 }));
      older.resolve(tree(node('c1')));
      await waitFor(() => expect(getPostComments).toHaveBeenCalledTimes(2));
      newer.resolve(tree(node('c1', { vote: 1, score: 1 })));
      await reload;

      expect(getPostComments).toHaveBeenLastCalledWith('+controversy');
    });

    it('leaves out the comments of a user hidden meanwhile', async () => {
      const { dispatch, getPostComments, nextResponse, comments } = setup();
      const older = nextResponse();
      const newer = nextResponse();

      const reload = dispatch(reloadComments());
      dispatch(hideUser(author));
      older.resolve(tree(node('c1')));
      await waitFor(() => expect(getPostComments).toHaveBeenCalledTimes(2));
      newer.resolve(tree(node('c1')));
      await reload;

      expect(comments().topComments).toEqual([]);
    });

    it('does not put the order of an earlier sort change back', async () => {
      const { dispatch, getPostComments, nextResponse } = setup();
      const first = nextResponse();
      const second = nextResponse();
      const third = nextResponse();

      const earlier = dispatch(updateSorting('+controversy'));
      const later = dispatch(updateSorting('-score'));
      second.resolve(tree(node('c1', { score: 2 })));
      await later;
      // the answer to the earlier choice comes last and is in an order the reader has left
      first.resolve(tree(node('c1', { score: 1 })));
      await waitFor(() => expect(getPostComments).toHaveBeenCalledTimes(3));
      third.resolve(tree(node('c1', { score: 2 })));
      await earlier;

      expect(getPostComments.mock.calls).toEqual([['+controversy'], ['-score'], ['-score']]);
    });

    it('does not let an older load overwrite a newer one', async () => {
      const { dispatch, getPostComments, nextResponse, comments } = setup();
      const first = nextResponse();
      const second = nextResponse();
      const third = nextResponse();

      const olderLoad = dispatch(reloadComments());
      const newerLoad = dispatch(reloadComments());
      second.resolve(tree(node('c1', { vote: 1, score: 1 })));
      await newerLoad;
      // the answer to the load that started first comes last
      first.resolve(tree(node('c1')));

      await waitFor(() => expect(getPostComments).toHaveBeenCalledTimes(3));
      expect(comments().allComments.c1.vote).toBe(1);

      third.resolve(tree(node('c1', { vote: 1, score: 1 })));
      await olderLoad;

      expect(comments().allComments.c1.vote).toBe(1);
    });
  });
});
