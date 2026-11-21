import * as api from 'common/api';
import type { Tree, Comment, Node, Sorting } from 'common/types';
import { CommentMode } from 'common/types';

import type { StoreAction, StoreState } from '../index';
import { setPostInfo } from '../post-info/actions';
import { filterTree } from './comment-tree';
import type { COMMENT_MODE_SET_ACTION, COMMENT_PATCH_ACTION, COMMENTS_EDIT_ACTION } from './types';
import {
  COMMENTS_SET,
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

/** sets comments, and put pinned comments in cache */
export const setComments =
  (comments: Node[]): StoreAction =>
  (dispatch) => {
    dispatch({
      type: COMMENTS_SET,
      comments,
    });
  };

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

/** requests the comments and puts them into the store, with or without the loading state */
const requestComments =
  (withPreloader: boolean): StoreAction<Promise<Tree>> =>
  async (dispatch, getState) => {
    if (withPreloader) dispatch({ type: COMMENTS_REQUEST_FETCHING });
    let data: Tree;
    let requestedFor: StoreState['comments']['allComments'];
    // a response is taken only when the store still holds the comments it was requested for. A
    // comment posted, edited or voted on while the request was in flight is newer than the
    // response, which would take the post off the list and put the vote back, so the comments are
    // requested again. Each request asks for the sort the store holds by then, which a change of
    // sort has set before it loads the comments
    do {
      requestedFor = getState().comments.allComments;
      data = await api.getPostComments(getState().comments.sort);
    } while (getState().comments.allComments !== requestedFor);
    if (withPreloader) dispatch({ type: COMMENTS_REQUEST_SUCCESS });
    const hiddenUsersIds = Object.keys(getState().hiddenUsers);
    if (hiddenUsersIds.length > 0) {
      data.comments = filterTree(data.comments, (node) => hiddenUsersIds.indexOf(node.comment.user.id) === -1);
    }

    dispatch(setComments(data.comments));
    dispatch(setPostInfo(data.info));

    return data;
  };

/** fetches comments from server; a preloader takes the place of the list until they arrive */
export const fetchComments = (): StoreAction<Promise<Tree>> => requestComments(true);

/**
 * fetches comments again and keeps the loaded ones on screen until the new tree replaces them.
 * With a preloader in their place the widget shrinks for the length of the request, and a reader
 * far down a long thread is left at another place on the host page when the list comes back
 */
export const reloadComments = (): StoreAction<Promise<Tree>> => requestComments(false);

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
