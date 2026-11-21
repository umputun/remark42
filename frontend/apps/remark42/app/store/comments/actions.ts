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
  (sort: Sorting | undefined, withPreloader: boolean): StoreAction<Promise<Tree>> =>
  async (dispatch, getState) => {
    const { hiddenUsers, comments } = getState();
    const hiddenUsersIds = Object.keys(hiddenUsers);
    if (withPreloader) dispatch({ type: COMMENTS_REQUEST_FETCHING });
    const data = await api.getPostComments(sort || comments.sort);
    if (withPreloader) dispatch({ type: COMMENTS_REQUEST_SUCCESS });
    if (hiddenUsersIds.length > 0) {
      data.comments = filterTree(data.comments, (node) => hiddenUsersIds.indexOf(node.comment.user.id) === -1);
    }

    dispatch(setComments(data.comments));
    dispatch(setPostInfo(data.info));

    return data;
  };

/** fetches comments from server; a preloader takes the place of the list until they arrive */
export const fetchComments = (sort?: Sorting): StoreAction<Promise<Tree>> => requestComments(sort, true);

/**
 * fetches comments again and keeps the loaded ones on screen until the new tree replaces them.
 * With a preloader in their place the widget shrinks for the length of the request, and a reader
 * far down a long thread is left at another place on the host page when the list comes back
 */
export const reloadComments = (): StoreAction<Promise<Tree>> => requestComments(undefined, false);

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
      await dispatch(fetchComments(sort));
      setItem(LS_SORT_KEY, sort);
    } catch (e) {
      dispatch({ type: COMMENTS_SET_SORT, payload: prevSort });
    }
  };
}
