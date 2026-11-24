import * as api from 'common/api';
import type { Tree, Comment, Node, Sorting } from 'common/types';
import { CommentMode } from 'common/types';

import type { StoreAction, StoreState } from '../index';
import { setPostInfo } from '../post-info/actions';
import { countComments, dropLoadedThreads, filterTree } from './comment-tree';
import type { COMMENT_MODE_SET_ACTION, COMMENT_PATCH_ACTION, COMMENTS_EDIT_ACTION, CommentsPage } from './types';
import {
  COMMENTS_SET,
  COMMENTS_APPEND_PAGE,
  COMMENTS_PAGE_SET,
  COMMENT_MODE_SET,
  COMMENTS_APPEND,
  COMMENTS_EDIT,
  COMMENTS_SET_SORT,
  COMMENTS_REQUEST_FETCHING,
  COMMENTS_REQUEST_SUCCESS,
  COMMENT_PATCH,
} from './types';
import { setItem } from 'common/local-storage';
import { LS_SORT_KEY } from 'common/constants';
import { commentsPageSize } from 'common/settings';

/** sets comments, and put pinned comments in cache */
export const setComments =
  (comments: Node[], pinned?: Comment[]): StoreAction =>
  (dispatch) => {
    dispatch({
      type: COMMENTS_SET,
      comments,
      pinned,
    });
  };

function setPage(page: CommentsPage) {
  return { type: COMMENTS_PAGE_SET, page } as const;
}

/** counts requests of the thread from its start, so that a page asked for before one of them is dropped */
let threadRequests = 0;

/** leaves out the threads, replies included, of the users the reader hid */
function hideInTree(tree: Node[], state: StoreState): Node[] {
  const hidden = Object.keys(state.hiddenUsers);
  return hidden.length === 0 ? tree : filterTree(tree, (node) => hidden.indexOf(node.comment.user.id) === -1);
}

/** appends comment to tree */
export const addComment =
  (text: string, title: string, pid?: Comment['id']): StoreAction<Promise<void>> =>
  async (dispatch) => {
    const comment = await api.addComment({ text, title, pid });
    dispatch({ type: COMMENTS_APPEND, pid: pid || null, comment });
  };

function editComments(comment: Comment): COMMENTS_EDIT_ACTION {
  return { type: COMMENTS_EDIT, comment };
}

export function patchComment(patch: Pick<Comment, 'id'> & Partial<Comment>): COMMENT_PATCH_ACTION {
  return { type: COMMENT_PATCH, patch };
}

/** edits comment in tree */
export const updateComment =
  (id: Comment['id'], text: string): StoreAction<Promise<void>> =>
  async (dispatch) => {
    const comment = await api.updateComment({ id, text });
    dispatch(editComments(comment));
  };

export const approveComment =
  (id: Comment['id'], text: string, approved: boolean): StoreAction<Promise<void>> =>
  async (dispatch) => {
    const comment = await api.approveComment({ id, text, approved });
    dispatch(patchComment({ ...comment, pendingApproval: comment.pendingApproval }));
  };

/** edits comment in tree */
export const setPinState =
  (id: Comment['id'], value: boolean): StoreAction<Promise<void>> =>
  async (dispatch, getState) => {
    if (value) {
      await api.pinComment(id);
    } else {
      await api.unpinComment(id);
    }
    let comment = getState().comments.allComments[id];
    comment = { ...comment, pin: value, edit: { summary: '', time: new Date().toISOString() } };
    dispatch(editComments(comment));
  };

/** edits comment in tree */
export const removeComment =
  (id: Comment['id']): StoreAction<Promise<void>> =>
  async (dispatch, getState) => {
    const user = getState().user;
    if (!user) return;
    if (user.admin) {
      await api.removeComment(id);
    } else {
      await api.removeMyComment(id);
    }
    let comment = getState().comments.allComments[id];
    comment = { ...comment, delete: true, edit: { summary: '', time: new Date().toISOString() } };
    dispatch(editComments(comment));
  };

/** how many comments to ask for from the start of the thread, measured as the request goes out; 0 asks for all */
type Limit = (state: StoreState) => number;

const firstPage: Limit = () => commentsPageSize;

/**
 * the loaded part of the thread: all of it when it was loaded to its end, otherwise as many comments
 * as are loaded and a page at least
 */
const loadedPart: Limit = ({ comments: { page } }) => {
  if (!commentsPageSize || (page.loaded > 0 && page.countLeft === 0)) return 0;
  return Math.max(commentsPageSize, page.loaded);
};

const isUndeleted = (comment: Comment) => !comment.delete;

/** requests the thread from its start and puts it into the store, with or without the loading state */
const requestComments =
  (withPreloader: boolean, limit: Limit = firstPage): StoreAction<Promise<Tree>> =>
  async (dispatch, getState) => {
    threadRequests++;
    if (withPreloader) dispatch({ type: COMMENTS_REQUEST_FETCHING });
    let data: Tree;
    let requestedFor: StoreState['comments']['allComments'];
    // a response is taken only when the store still holds the comments it was requested for. A
    // comment posted, edited or voted on while the request was in flight is newer than the
    // response, which would take the post off the list and put the vote back, so the comments are
    // requested again. Each request asks for the sort the store holds by then, which a change of
    // sort has set before it loads the comments, and for the loaded part as it is by then, which a
    // page that came meanwhile has grown
    do {
      requestedFor = getState().comments.allComments;
      data = await api.getPostComments(getState().comments.sort, { limit: limit(getState()) });
    } while (getState().comments.allComments !== requestedFor);
    if (withPreloader) dispatch({ type: COMMENTS_REQUEST_SUCCESS });
    const countLeft = data.info.count_left || 0;
    const hidden = Object.keys(getState().hiddenUsers);
    // the server's list stands in for the pins found in the tree only while part of the thread is
    // still to come, so a thread that came whole shows its pins in its own order, as it always did
    const pinned = countLeft > 0 ? data.pinned?.filter((comment) => hidden.indexOf(comment.user.id) === -1) : undefined;
    const page = {
      countLeft,
      lastComment: data.info.last_comment || '',
      loaded: countComments(data.comments),
      undeleted: countComments(data.comments, isUndeleted),
    };
    data.comments = hideInTree(data.comments, getState());

    dispatch(setComments(data.comments, pinned));
    dispatch(setPage(page));
    dispatch(setPostInfo(data.info));

    return data;
  };

/**
 * the pages a reload loads by itself while a thread that was on screen is missing from it. A thread
 * deleted since never comes back, and the server tells no deleted thread from one further down, so
 * past these pages the rest is left to Show more
 */
const MISSING_THREAD_PAGES = 2;

/**
 * requests the loaded part of the thread again, and then the pages that bring back the threads it
 * no longer holds. Comments added to the loaded threads since take room in it, and a page stops
 * before a thread that does not fit; under the sorts by activity, score or controversy a loaded
 * thread can also move behind the unloaded ones. So up to MISSING_THREAD_PAGES pages follow while a
 * thread that was on screen is missing and the thread has more
 */
const requestLoadedPart =
  (withPreloader: boolean): StoreAction<Promise<Tree>> =>
  async (dispatch, getState) => {
    // gathered on every attempt, since a page that came while a request was on its way is on screen too,
    // and kept from the attempts before, since another reload answering meanwhile can shorten the list
    const onScreen = new Set<Comment['id']>();
    const limit: Limit = (state) => {
      state.comments.topComments.forEach((id) => onScreen.add(id));
      return loadedPart(state);
    };
    const pending = dispatch(requestComments(withPreloader, limit));
    // taken as the request starts: a change of sort or another reload started later has a loaded part of its own
    const request = threadRequests;
    const data = await pending;
    // a thread hidden by the time of the check is not on screen, and no answer brings it back
    const isMissing = () => {
      const { page, topComments, allComments } = getState().comments;
      if (page.countLeft === 0) return false;
      const shown = new Set(topComments);
      return Array.from(onScreen).some((id) => !shown.has(id) && !allComments[id]?.hidden);
    };
    for (let pages = 0; pages < MISSING_THREAD_PAGES && request === threadRequests && isMissing(); pages++) {
      // a page the reader asked for meanwhile may bring the missing threads back by itself
      await moreComments;
      if (request !== threadRequests || !isMissing()) break;
      await dispatch(loadMoreComments());
    }
    return data;
  };

/**
 * fetches the first page of comments from server; a preloader takes the place of the list until
 * they arrive. With `keepLoaded`, the part of the thread that is loaded already
 */
export const fetchComments = ({ keepLoaded = false } = {}): StoreAction<Promise<Tree>> =>
  keepLoaded ? requestLoadedPart(true) : requestComments(true);

/**
 * fetches the loaded part of the thread again and keeps it on screen until the new tree replaces it.
 * With a preloader in its place the widget shrinks for the length of the request, and a reader
 * far down a long thread is left at another place on the host page when the list comes back
 */
export const reloadComments = (): StoreAction<Promise<Tree>> => requestLoadedPart(false);

/** the last call of loadMoreComments, which the next one waits for, so that two never ask for the same page */
let moreComments: Promise<void> = Promise.resolve();

/**
 * loads the threads after the loaded ones: a page of them, or with `rest` all that are left, which a
 * link to a comment not loaded yet needs. Calls run one after another, each from where the one
 * before it ended, and a request of the thread from its start, made meanwhile by a change of sort or
 * a reload, makes a call drop what it got.
 *
 * Every call either adds threads or leaves nothing to load:
 * - when the comment a page starts after is gone, deleted since, the server starts from the top and
 *   the page brings only loaded threads; the loaded part is then requested again with a page more,
 *   which carries on where it ended
 * - a page of hidden users' threads only shows nothing, so the next one follows
 * - at the end of the thread, a thread that moved ahead of the loaded ones while they were paged
 *   through, as a reply or a vote moves one under the sorts by activity, score or controversy, was
 *   in none of the pages. The server's count then exceeds the undeleted comments received, and the
 *   thread is requested again whole. Comments deleted from the loaded part meanwhile lower that
 *   count but not the comments received, so a skipped thread no bigger than the deletions goes
 *   unnoticed until a reload
 *
 * A reload whose answer lands while a call waits for its page moves the end of the loaded part,
 * and the call then asks again from where the reload ended.
 */
export const loadMoreComments =
  ({ rest = false } = {}): StoreAction<Promise<void>> =>
  (dispatch) => {
    const call = moreComments.then(() => dispatch(requestMoreComments(rest)));
    moreComments = call.catch(() => undefined);
    return call;
  };

const requestMoreComments =
  (rest: boolean): StoreAction<Promise<void>> =>
  async (dispatch, getState) => {
    for (;;) {
      const { page, sort } = getState().comments;
      if (page.countLeft === 0) return;
      const request = threadRequests;
      const data = await api.getPostComments(sort, {
        limit: rest ? 0 : commentsPageSize,
        offsetId: page.lastComment,
      });
      if (request !== threadRequests) return;
      if (getState().comments.page !== page) continue;

      const added = dropLoadedThreads(data.comments, getState().comments.topComments);
      const countLeft = data.info.count_left || 0;
      if (added.length === 0 && countLeft > 0) {
        await dispatch(requestComments(false, () => page.loaded + commentsPageSize));
        return;
      }

      const shown = hideInTree(added, getState());
      const next = {
        countLeft,
        lastComment: data.info.last_comment || page.lastComment,
        loaded: page.loaded + countComments(added),
        undeleted: page.undeleted + countComments(added, isUndeleted),
      };
      dispatch({ type: COMMENTS_APPEND_PAGE, comments: shown });
      dispatch(setPage(next));
      dispatch(setPostInfo(data.info));

      if (countLeft === 0 && data.info.count > next.undeleted) {
        await dispatch(requestComments(false, () => 0));
        return;
      }
      if (shown.length > 0 || countLeft === 0) return;
    }
  };

/** sets mode for comment, either reply or edit */
export const setCommentMode =
  (mode: StoreState['comments']['activeComment']): StoreAction =>
  (dispatch) => {
    if (mode !== null && mode.state === CommentMode.None) {
      mode = null;
    }
    dispatch(unsetCommentMode(mode));
  };

/** unsets comment mode */
export function unsetCommentMode(mode: StoreState['comments']['activeComment'] = null) {
  return {
    type: COMMENT_MODE_SET,
    mode,
  } as COMMENT_MODE_SET_ACTION;
}

export function updateSorting(sort: Sorting): StoreAction {
  return async (dispatch, getState) => {
    const { sort: prevSort } = getState().comments;
    dispatch({ type: COMMENTS_SET_SORT, payload: sort });

    try {
      await dispatch(fetchComments());
      setItem(LS_SORT_KEY, sort);
    } catch (e) {
      dispatch({ type: COMMENTS_SET_SORT, payload: prevSort });
    }
  };
}
