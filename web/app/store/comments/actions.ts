import api from '@app/common/api';
import { Node, Tree, Comment, Sorting } from '@app/common/types';

import { StoreAction } from '../index';
import { POST_INFO_SET } from '../post_info/types';
import {
  getPinnedComments,
  addComment as uAddComment,
  replaceComment as uReplaceComment,
  removeComment as uRemoveComment,
  setCommentPin as uSetCommentPin,
} from './utils';
import { COMMENTS_SET, PINNED_COMMENTS_SET } from './types';

/** sets comments, and put pinned comments in cache */
export const setComments = (comments: Node[]): StoreAction<void> => dispatch => {
  dispatch({
    type: COMMENTS_SET,
    comments,
  });
  dispatch({
    type: PINNED_COMMENTS_SET,
    comments: getPinnedComments(comments),
  });
};

/** appends comment to tree */
export const addComment = (text: string, title: string, pid?: Comment['id']): StoreAction<Promise<void>> => async (
  dispatch,
  getState
) => {
  const comment = await api.addComment({ text, title, pid });
  const comments = getState().comments;
  dispatch(setComments(uAddComment(comments, comment)));
};

/** edits comment in tree */
export const updateComment = (id: Comment['id'], text: string): StoreAction<Promise<void>> => async (
  dispatch,
  getState
) => {
  const comment = await api.updateComment({ id, text });
  const comments = getState().comments;
  dispatch(setComments(uReplaceComment(comments, comment)));
};

/** edits comment in tree */
export const putVote = (id: Comment['id'], value: number): StoreAction<Promise<void>> => async (dispatch, getState) => {
  await api.putCommentVote({ id, value });
  const updatedComment = await api.getComment({ id });
  const comments = getState().comments;
  dispatch(setComments(uReplaceComment(comments, updatedComment)));
};

/** edits comment in tree */
export const setPinState = (id: Comment['id'], value: boolean): StoreAction<Promise<void>> => async (
  dispatch,
  getState
) => {
  if (value) {
    await api.pinComment(id);
  } else {
    await api.unpinComment(id);
  }
  const comments = getState().comments;
  dispatch(setComments(uSetCommentPin(comments, id, value)));
};

/** edits comment in tree */
export const removeComment = (id: Comment['id']): StoreAction<Promise<void>> => async (dispatch, getState) => {
  const user = getState().user;
  if (!user) return;
  if (user.admin) {
    await api.removeComment({ id });
  } else {
    await api.removeMyComment({ id });
  }
  const comments = getState().comments;
  dispatch(setComments(uRemoveComment(comments, id)));
};

/** fetches comments from server */
export const fetchComments = (sort: Sorting): StoreAction<Promise<Tree>> => async dispatch => {
  const data = await api.getPostComments(sort);
  dispatch(setComments(data.comments));
  dispatch({
    type: POST_INFO_SET,
    info: data.info,
  });
  return data;
};

/** set state of post: readonly or not */
export const setCommentsReadOnlyState = (state: boolean): StoreAction<Promise<boolean>> => async (
  dispatch,
  getState
) => {
  await (!state ? api.enableComments() : api.disableComments());
  const storeState = getState();
  dispatch({
    type: POST_INFO_SET,
    info: { ...storeState.info, read_only: state },
  });
  return state;
};

/** toggles state of post: readonly or not */
export const toggleCommentsReadOnlyState = (): StoreAction<Promise<boolean>> => async (dispatch, getState) => {
  const storeState = getState();
  const state = !storeState.info.read_only!;
  await (state ? api.enableComments() : api.disableComments());
  dispatch({
    type: POST_INFO_SET,
    info: { ...storeState.info, read_only: !state },
  });
  return !state;
};
