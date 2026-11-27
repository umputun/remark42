import { combineReducers, createStore } from 'store/create-store';
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
  loadMoreComments,
  patchComment,
  reloadComments,
  setComments,
  updateComment,
  updateSorting,
} from './actions';
import {
  COMMENTS_PAGE_SET,
  COMMENTS_REQUEST_FETCHING,
  COMMENTS_REQUEST_SUCCESS,
  COMMENTS_SET,
  COMMENTS_SET_SORT,
} from './types';

/** the page of a thread that came whole */
const wholeThread = { countLeft: 0, lastComment: '', loaded: 0, undeleted: 0 };

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
    jest.spyOn(api, 'getPostComments').mockResolvedValueOnce({ comments: [], info: { url: '', count: 0 } });

    await store.dispatch(updateSorting(newSort));

    const [setCommentAction] = store.getActions();

    expect(setCommentAction).toEqual({ type: COMMENTS_SET_SORT, payload: newSort });
    expect(localStorage.setItem).toHaveBeenCalledWith(LS_SORT_KEY, newSort);
  });

  describe('loading the thread', () => {
    const info = { url: 'https://example.com/post', count: 0 };
    const state = {
      comments: { sort: '+active', page: wholeThread, topComments: [], allComments: {} },
      hiddenUsers: {},
    };

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
        { type: COMMENTS_PAGE_SET, page: wholeThread },
        { type: POST_INFO_SET, info },
      ]);
    });

    it('leaves the loaded comments on screen when reloading', async () => {
      const store = mockStore(state);

      await store.dispatch(reloadComments());

      expect(store.getActions()).toEqual([
        { type: COMMENTS_SET, comments: [] },
        { type: COMMENTS_PAGE_SET, page: wholeThread },
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
      const store = createStore(combineReducers(rootProvider));
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

      expect(getPostComments).toHaveBeenLastCalledWith('+controversy', { limit: 100 });
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

      expect(getPostComments.mock.calls).toEqual([
        ['+controversy', { limit: 100 }],
        ['-score', { limit: 100 }],
        ['-score', { limit: 100 }],
      ]);
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

  // the widget's own reducers, so that what a page adds to the thread is what the reader sees
  describe('a thread loaded in pages', () => {
    // `count` is the server's count of the thread's comments that are not deleted
    const tree = (
      comments: Node[],
      page: { count_left?: number; last_comment?: string; count?: number } = {},
      pinned?: Comment[]
    ) => ({ comments, info: { url: 'https://example.com/post', count: 0, ...page }, pinned }) as Tree;

    function setup() {
      const store = createStore(combineReducers(rootProvider));
      const dispatch: StoreDispatch = store.dispatch;
      const getPostComments = jest.spyOn(api, 'getPostComments').mockReset();

      return { dispatch, getPostComments, comments: () => store.getState().comments };
    }

    it('asks for a page and keeps where it ends', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(
        tree([{ ...node('c1'), replies: [node('r1')] }, node('c2')], { count_left: 5, last_comment: 'c2' })
      );

      await dispatch(fetchComments());

      expect(getPostComments).toHaveBeenCalledWith('-active', { limit: 100 });
      expect(comments().page).toEqual({ countLeft: 5, lastComment: 'c2', loaded: 3, undeleted: 3 });
    });

    it('appends the next page after the last loaded thread, leaving out the threads already loaded', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('c1'), node('c2')], { count_left: 2, last_comment: 'c2' }));
      await dispatch(fetchComments());
      // the order moved between the requests, and c2 came again
      getPostComments.mockResolvedValueOnce(
        tree([node('c2'), { ...node('c3'), replies: [node('r3')] }], { count_left: 0, last_comment: 'c3' })
      );

      await dispatch(loadMoreComments());

      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 100, offsetId: 'c2' });
      expect(comments().topComments).toEqual(['c1', 'c2', 'c3']);
      expect(comments().childComments.c3).toEqual(['r3']);
      expect(comments().allComments.r3.id).toBe('r3');
      // the repeated c2 is not counted twice
      expect(comments().page).toEqual({ countLeft: 0, lastComment: 'c3', loaded: 4, undeleted: 4 });
    });

    it('requests the loaded part again with a page more when a page brings only loaded threads', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('c1'), node('c2')], { count_left: 1, last_comment: 'c2' }));
      await dispatch(fetchComments());
      // c2 was deleted since, so the server started again from the top
      getPostComments.mockResolvedValueOnce(tree([node('c1')], { count_left: 1, last_comment: 'c1' }));
      getPostComments.mockResolvedValueOnce(tree([node('c1'), node('c3')], { count_left: 0, last_comment: 'c3' }));

      await dispatch(loadMoreComments());

      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 102 });
      expect(comments().topComments).toEqual(['c1', 'c3']);
      expect(comments().page.countLeft).toBe(0);
    });

    it('drops a page that a request of the thread from its start overtook', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('c1')], { count_left: 1, last_comment: 'c1' }));
      await dispatch(fetchComments());
      const page = deferred<Tree>();
      getPostComments.mockReturnValueOnce(page.promise);
      getPostComments.mockResolvedValueOnce(tree([node('c9')], { count_left: 1, last_comment: 'c9' }));

      const more = dispatch(loadMoreComments());
      await waitFor(() => expect(getPostComments).toHaveBeenCalledTimes(2));
      await dispatch(updateSorting('+time'));
      page.resolve(tree([node('c2')], { count_left: 0, last_comment: 'c2' }));
      await more;

      expect(comments().topComments).toEqual(['c9']);
      expect(comments().page).toEqual({ countLeft: 1, lastComment: 'c9', loaded: 1, undeleted: 1 });
    });

    it('reloads as many comments as are loaded', async () => {
      const { dispatch, getPostComments } = setup();
      // the loaded part comes back whole, up to c7 where it ended
      getPostComments.mockResolvedValue(tree([node('c1'), node('c7')], { count_left: 4, last_comment: 'c7' }));
      const deepInTheThread = () =>
        dispatch({ type: COMMENTS_PAGE_SET, page: { countLeft: 4, lastComment: 'c7', loaded: 250, undeleted: 250 } });

      deepInTheThread();
      await dispatch(reloadComments());
      deepInTheThread();
      await dispatch(fetchComments({ keepLoaded: true }));
      deepInTheThread();
      // the first page of a new order, or of a thread opened again
      await dispatch(fetchComments());

      expect(getPostComments.mock.calls).toEqual([
        ['-active', { limit: 250 }],
        ['-active', { limit: 250 }],
        ['-active', { limit: 100 }],
      ]);
    });

    it('brings back the thread that ended the loaded part when comments added since push it out of a reload', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('c1'), node('c2')], { count_left: 1, last_comment: 'c2' }));
      await dispatch(fetchComments());
      // a reply to c1 takes the room c2 had, and the server stops before c2
      getPostComments.mockResolvedValueOnce(
        tree([{ ...node('c1'), replies: [node('r1')] }], { count_left: 2, last_comment: 'c1' })
      );
      getPostComments.mockResolvedValueOnce(tree([node('c2')], { count_left: 1, last_comment: 'c2' }));

      await dispatch(reloadComments());

      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 100, offsetId: 'c1' });
      expect(comments().topComments).toEqual(['c1', 'c2']);
      expect(comments().page.countLeft).toBe(1);
    });

    it('leaves the rest of the thread to Show more when a reload does not find a thread deleted since', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('c1'), node('c2')], { count_left: 50, last_comment: 'c2' }));
      await dispatch(fetchComments());
      // c2 is deleted with no replies, so no answer has it, and every page leaves more to load
      getPostComments.mockResolvedValueOnce(tree([node('c1'), node('c3')], { count_left: 49, last_comment: 'c3' }));
      getPostComments.mockResolvedValueOnce(tree([node('c4')], { count_left: 48, last_comment: 'c4' }));
      getPostComments.mockResolvedValueOnce(tree([node('c5')], { count_left: 47, last_comment: 'c5' }));
      getPostComments.mockResolvedValue(tree([node('c6')], { count_left: 46, last_comment: 'c6' }));

      await dispatch(reloadComments());

      // the reload and two pages after it
      expect(getPostComments).toHaveBeenCalledTimes(4);
      expect(comments().topComments).toEqual(['c1', 'c3', 'c4', 'c5']);
      expect(comments().page.countLeft).toBe(47);
    });

    it('brings back a thread that moved behind the unloaded ones on a reload', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('a'), node('b')], { count_left: 2, last_comment: 'b' }));
      await dispatch(fetchComments());
      // a reply moved b ahead, and with it b fills the reload, so a is left out though b, the last loaded thread, is back
      getPostComments.mockResolvedValueOnce(
        tree([{ ...node('b'), replies: [node('rb')] }], { count_left: 3, last_comment: 'b' })
      );
      getPostComments.mockResolvedValueOnce(tree([node('a')], { count_left: 2, last_comment: 'a' }));

      await dispatch(reloadComments());

      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 100, offsetId: 'b' });
      expect(comments().topComments).toEqual(['b', 'a']);
    });

    it('leaves the pages of a new sort to the reader when a reload answers after the sort changed', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('a'), node('b')], { count_left: 2, last_comment: 'b' }));
      await dispatch(fetchComments());
      const reloaded = deferred<Tree>();
      const sorted = deferred<Tree>();
      getPostComments.mockReturnValueOnce(reloaded.promise);
      getPostComments.mockReturnValueOnce(sorted.promise);
      getPostComments.mockResolvedValue(tree([node('z')], { count_left: 0, last_comment: 'z' }));

      const reload = dispatch(reloadComments());
      const sort = dispatch(updateSorting('+time'));
      await waitFor(() => expect(getPostComments).toHaveBeenCalledTimes(3));
      // the reload comes back short of b, but the thread is now asked for in another order
      reloaded.resolve(tree([node('a')], { count_left: 3, last_comment: 'a' }));
      await reload;
      sorted.resolve(tree([node('x')], { count_left: 3, last_comment: 'x' }));
      await sort;

      // the sort's own request is sent again, since the reload changed the comments while it was on its way,
      // but no page of the new order follows the old cursor
      expect(getPostComments.mock.calls.filter(([, page]) => page?.offsetId)).toEqual([]);
      expect(comments().topComments).toEqual(['z']);
    });

    it('asks again from where a reload ended when it lands while a page is on its way', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('a'), node('b')], { count_left: 3, last_comment: 'b' }));
      await dispatch(fetchComments());
      const reloaded = deferred<Tree>();
      const page = deferred<Tree>();
      getPostComments.mockReturnValueOnce(reloaded.promise);
      getPostComments.mockReturnValueOnce(page.promise);
      getPostComments.mockResolvedValueOnce(tree([node('b')], { count_left: 3, last_comment: 'b' }));
      getPostComments.mockResolvedValue(tree([node('zz')], { count_left: 0 }));

      const reload = dispatch(reloadComments());
      const click = dispatch(loadMoreComments());
      await waitFor(() => expect(getPostComments).toHaveBeenCalledTimes(3));
      // a reply to a took the room b had
      reloaded.resolve(tree([{ ...node('a'), replies: [node('ra')] }], { count_left: 4, last_comment: 'a' }));
      await waitFor(() => expect(comments().topComments).toEqual(['a']));
      // the page the click asked for starts after b, which the reload no longer holds
      page.resolve(tree([node('c')], { count_left: 2, last_comment: 'c' }));
      await Promise.all([reload, click]);

      // the click's own page brought b back, so the reload asks for nothing more
      expect(getPostComments).toHaveBeenCalledTimes(4);
      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 100, offsetId: 'a' });
      expect(comments().topComments).toEqual(['a', 'b']);
    });

    it('brings back on a reload the threads of a page that came while the reload was on its way', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('a'), node('b')], { count_left: 3, last_comment: 'b' }));
      await dispatch(fetchComments());
      const reloaded = deferred<Tree>();
      getPostComments.mockReturnValueOnce(reloaded.promise);
      getPostComments.mockResolvedValueOnce(tree([node('c')], { count_left: 2, last_comment: 'c' }));
      // the reload is sent again for the grown part, and a reply to a leaves c out of it
      getPostComments.mockResolvedValueOnce(
        tree([{ ...node('a'), replies: [node('ra')] }, node('b')], { count_left: 3, last_comment: 'b' })
      );
      getPostComments.mockResolvedValueOnce(tree([node('c')], { count_left: 2, last_comment: 'c' }));

      const reload = dispatch(reloadComments());
      await dispatch(loadMoreComments());
      reloaded.resolve(tree([node('a'), node('b')], { count_left: 3, last_comment: 'b' }));
      await reload;

      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 100, offsetId: 'b' });
      expect(comments().topComments).toEqual(['a', 'b', 'c']);
    });

    it('does not look for the thread of a user hidden since on a reload', async () => {
      const { dispatch, getPostComments, comments } = setup();
      const troll = { ...author, id: 'troll' };
      getPostComments.mockResolvedValueOnce(
        tree([node('a', { user: troll }), node('b')], { count_left: 5, last_comment: 'b' })
      );
      await dispatch(fetchComments());
      dispatch(hideUser(troll));
      getPostComments.mockResolvedValue(
        tree([node('a', { user: troll }), node('b')], { count_left: 5, last_comment: 'b' })
      );

      await dispatch(reloadComments());

      expect(getPostComments).toHaveBeenCalledTimes(2);
      expect(comments().topComments).toEqual(['b']);
    });

    it('does not look for a thread hidden while the reload was on its way', async () => {
      const { dispatch, getPostComments, comments } = setup();
      const troll = { ...author, id: 'troll' };
      const page = tree([node('a', { user: troll }), node('b')], { count_left: 5, last_comment: 'b' });
      getPostComments.mockResolvedValueOnce(page);
      await dispatch(fetchComments());
      const reloaded = deferred<Tree>();
      getPostComments.mockReturnValueOnce(reloaded.promise);
      getPostComments.mockResolvedValue(page);

      const reload = dispatch(reloadComments());
      dispatch(hideUser(troll));
      reloaded.resolve(page);
      await reload;

      // the hide changed the comments, so the reload is sent again, and no page follows it
      expect(getPostComments).toHaveBeenCalledTimes(3);
      expect(getPostComments.mock.calls.filter(([, p]) => p?.offsetId)).toEqual([]);
      expect(comments().topComments).toEqual(['b']);
    });

    it('keeps looking for a thread that an earlier reload answering meanwhile left out', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('a'), node('b')], { count_left: 2, last_comment: 'b' }));
      await dispatch(fetchComments());
      const older = deferred<Tree>();
      const newer = deferred<Tree>();
      getPostComments.mockReturnValueOnce(older.promise);
      getPostComments.mockReturnValueOnce(newer.promise);
      // a reply to a fills the reload, and b is left out
      const short = tree([{ ...node('a'), replies: [node('ra')] }], { count_left: 3, last_comment: 'a' });
      getPostComments.mockResolvedValueOnce(short);
      getPostComments.mockResolvedValueOnce(tree([node('b')], { count_left: 2, last_comment: 'b' }));

      const first = dispatch(reloadComments());
      const second = dispatch(reloadComments());
      await waitFor(() => expect(getPostComments).toHaveBeenCalledTimes(3));
      older.resolve(short);
      await first;
      // the newer reload is sent again, since the older one changed the comments meanwhile
      newer.resolve(short);
      await second;

      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 100, offsetId: 'a' });
      expect(comments().topComments).toEqual(['a', 'b']);
    });

    it('reloads a thread loaded to its end whole', async () => {
      const { dispatch, getPostComments } = setup();
      dispatch({ type: COMMENTS_PAGE_SET, page: { countLeft: 0, lastComment: 'c7', loaded: 250, undeleted: 250 } });
      getPostComments.mockResolvedValue(tree([node('c1')]));

      await dispatch(reloadComments());

      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 0 });
    });

    it('requests a reload with the loaded part as it is when the request goes out again', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('c1'), node('c2')], { count_left: 2, last_comment: 'c2' }));
      await dispatch(fetchComments());
      const reloaded = deferred<Tree>();
      getPostComments.mockReturnValueOnce(reloaded.promise);
      getPostComments.mockResolvedValueOnce(tree([node('c3')], { count_left: 0, last_comment: 'c3' }));
      getPostComments.mockResolvedValueOnce(tree([node('c1'), node('c2'), node('c3')], { last_comment: 'c3' }));

      const reload = dispatch(reloadComments());
      // a page arrives while the reload is on its way, so the reload is sent again
      await dispatch(loadMoreComments());
      reloaded.resolve(tree([node('c1'), node('c2')], { count_left: 2, last_comment: 'c2' }));
      await reload;

      // the whole thread is loaded by the time the reload is sent again, so it asks for all of it
      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 0 });
      expect(comments().topComments).toEqual(['c1', 'c2', 'c3']);
    });

    it('requests the thread again whole when a thread moved ahead of the loaded ones while they were paged through', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(
        tree([node('c1'), node('c2')], { count_left: 2, last_comment: 'c2', count: 4 })
      );
      await dispatch(fetchComments());
      // a reply moved c3 ahead of c2, so the last page holds c4 alone
      getPostComments.mockResolvedValueOnce(tree([node('c4')], { count_left: 0, last_comment: 'c4', count: 5 }));
      getPostComments.mockResolvedValueOnce(
        tree([{ ...node('c3'), replies: [node('r3')] }, node('c1'), node('c2'), node('c4')], { count: 5 })
      );

      await dispatch(loadMoreComments());

      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 0 });
      expect(comments().topComments).toEqual(['c3', 'c1', 'c2', 'c4']);
    });

    it('goes on to the next page when one holds only threads of hidden users', async () => {
      const { dispatch, getPostComments, comments } = setup();
      dispatch(hideUser(author));
      const other = { ...author, id: 'other' };
      getPostComments.mockResolvedValueOnce(tree([node('c1', { user: other })], { count_left: 3, last_comment: 'c1' }));
      await dispatch(fetchComments());
      getPostComments.mockResolvedValueOnce(tree([node('c2'), node('c3')], { count_left: 1, last_comment: 'c3' }));
      getPostComments.mockResolvedValueOnce(tree([node('c4', { user: other })], { count_left: 0, last_comment: 'c4' }));

      await dispatch(loadMoreComments());

      expect(getPostComments).toHaveBeenCalledTimes(3);
      expect(comments().topComments).toEqual(['c1', 'c4']);
    });

    it('runs one load after another, so two never ask for the same page', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('c1')], { count_left: 3, last_comment: 'c1' }));
      await dispatch(fetchComments());
      const page = deferred<Tree>();
      getPostComments.mockReturnValueOnce(page.promise);
      getPostComments.mockResolvedValueOnce(tree([node('c3'), node('c4')], { last_comment: 'c4' }));

      const click = dispatch(loadMoreComments());
      const link = dispatch(loadMoreComments({ rest: true }));
      page.resolve(tree([node('c2')], { count_left: 2, last_comment: 'c2' }));
      await Promise.all([click, link]);

      expect(getPostComments.mock.calls.slice(1)).toEqual([
        ['-active', { limit: 100, offsetId: 'c1' }],
        ['-active', { limit: 0, offsetId: 'c2' }],
      ]);
      expect(comments().topComments).toEqual(['c1', 'c2', 'c3', 'c4']);
    });

    it('loads the whole rest of the thread when asked for it', async () => {
      const { dispatch, getPostComments, comments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('c1')], { count_left: 2, last_comment: 'c1' }));
      await dispatch(fetchComments());
      getPostComments.mockResolvedValueOnce(tree([node('c2'), node('c3')]));

      await dispatch(loadMoreComments({ rest: true }));

      expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 0, offsetId: 'c1' });
      expect(comments().topComments).toEqual(['c1', 'c2', 'c3']);
      expect(comments().page.countLeft).toBe(0);
    });

    it('requests nothing when the whole thread is loaded', async () => {
      const { dispatch, getPostComments } = setup();
      getPostComments.mockResolvedValueOnce(tree([node('c1')]));
      await dispatch(fetchComments());

      await dispatch(loadMoreComments());

      expect(getPostComments).toHaveBeenCalledTimes(1);
    });

    it('shows the pins of later pages from the list the first page carries', async () => {
      const { dispatch, getPostComments, comments } = setup();
      const pinned = comment('r8', { pid: 'c8', pin: true });
      getPostComments.mockResolvedValueOnce(tree([node('c1')], { count_left: 3, last_comment: 'c1' }, [pinned]));

      await dispatch(fetchComments());

      expect(comments().pinnedComments).toEqual(['r8']);
      expect(comments().allComments.r8).toEqual(pinned);
      // the thread itself still holds only the first page
      expect(comments().topComments).toEqual(['c1']);
    });

    it('keeps the pins of a thread that came whole in the order of the thread', async () => {
      const { dispatch, getPostComments, comments } = setup();
      const newer = node('c2', { pin: true });
      const older = node('c1', { pin: true });
      // newest first, while the server lists pins oldest first
      getPostComments.mockResolvedValueOnce(tree([newer, older], {}, [older.comment, newer.comment]));

      await dispatch(fetchComments());

      expect(comments().pinnedComments).toEqual(['c2', 'c1']);
    });

    it('leaves out the threads and pins of hidden users on every page', async () => {
      const { dispatch, getPostComments, comments } = setup();
      dispatch(hideUser(author));
      const other = { ...author, id: 'other' };
      getPostComments.mockResolvedValueOnce(
        tree([node('c1', { user: other }), node('c2')], { count_left: 2, last_comment: 'c2' }, [
          comment('c9', { pin: true }),
        ])
      );
      await dispatch(fetchComments());
      getPostComments.mockResolvedValueOnce(tree([node('c3'), node('c4', { user: other })], { count_left: 0 }));

      await dispatch(loadMoreComments());

      expect(comments().topComments).toEqual(['c1', 'c4']);
      expect(comments().pinnedComments).toEqual([]);
      // the hidden threads were loaded all the same, so the next page starts after them
      expect(comments().page.loaded).toBe(4);
    });
  });
});
