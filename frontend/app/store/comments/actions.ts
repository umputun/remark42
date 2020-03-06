import api from '@app/common/api';
import { Tree, Comment, CommentMode, Node, Sorting } from '@app/common/types';

import { StoreAction, StoreState } from '../index';
import { POST_INFO_SET } from '../post_info/types';
import { filterTree } from './utils';
import { COMMENTS_SET, COMMENT_MODE_SET, COMMENTS_APPEND, COMMENTS_EDIT } from './types';

/** sets comments, and put pinned comments in cache */
export const setComments = (comments: Node[]): StoreAction<void> => dispatch => {
  dispatch({
    type: COMMENTS_SET,
    comments,
  });
};

/** appends comment to tree */
export const addComment = (
  text: string,
  title: string,
  pid?: Comment['id']
): StoreAction<Promise<void>> => async dispatch => {
  const comment = await api.addComment({ text, title, pid });
  dispatch({ type: COMMENTS_APPEND, pid: pid || null, comment });
};

/** edits comment in tree */
export const updateComment = (id: Comment['id'], text: string): StoreAction<Promise<void>> => async dispatch => {
  const comment = await api.updateComment({ id, text });
  dispatch({ type: COMMENTS_EDIT, comment });
};

/** edits comment in tree */
export const putVote = (id: Comment['id'], value: number): StoreAction<Promise<void>> => async dispatch => {
  await api.putCommentVote({ id, value });
  const comment = await api.getComment(id);
  dispatch({ type: COMMENTS_EDIT, comment });
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
  let comment = getState().comments[id];
  comment = { ...comment, pin: value, edit: { summary: '', time: new Date().toISOString() } };
  dispatch({ type: COMMENTS_EDIT, comment });
};

/** edits comment in tree */
export const removeComment = (id: Comment['id']): StoreAction<Promise<void>> => async (dispatch, getState) => {
  const user = getState().user;
  if (!user) return;
  if (user.admin) {
    await api.removeComment(id);
  } else {
    await api.removeMyComment(id);
  }
  let comment = getState().comments[id];
  comment = { ...comment, delete: true, edit: { summary: '', time: new Date().toISOString() } };
  dispatch({ type: COMMENTS_EDIT, comment });
};

/** fetches comments from server */
export const fetchComments = (sort: Sorting): StoreAction<Promise<Tree>> => async (dispatch, getState) => {
  const { hiddenUsers } = getState();
  const hiddenUsersIds = Object.keys(hiddenUsers);
  const data = await api.getPostComments(sort);

  if (hiddenUsersIds.length > 0) {
    data.comments = filterTree(data.comments, node => hiddenUsersIds.indexOf(node.comment.user.id) === -1);
  }

  dispatch(setComments(data.comments));
  dispatch({
    type: POST_INFO_SET,
    info: data.info,
  });

  return data;
};

/** sets mode for comment, either reply or edit */
export const setCommentMode = (mode: StoreState['activeComment']): StoreAction<void> => dispatch => {
  if (mode !== null && mode.state === CommentMode.None) {
    mode = null;
  }
  dispatch({
    type: COMMENT_MODE_SET,
    mode,
  });
};

/** unsets comment mode */
export const unsetCommentMode = (): StoreAction<void> => dispatch => {
  dispatch({
    type: COMMENT_MODE_SET,
    mode: null,
  });
};
