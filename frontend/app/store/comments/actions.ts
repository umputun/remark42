import api from '@app/common/api';
import { Tree, Comment, Sorting, CommentMode } from '@app/common/types';

import { StoreAction, StoreState } from '../index';
import { POST_INFO_SET } from '../post_info/types';
import {
  getPinnedComments,
  addComment as uAddComment,
  replaceComment as uReplaceComment,
  removeComment as uRemoveComment,
  setCommentPin as uSetCommentPin,
  filterTree,
  flattenTree,
  mapTreeIfID,
  mapTree,
} from './utils';
import { COMMENTS_SET, PINNED_COMMENTS_SET, COMMENT_MODE_SET } from './types';

/** sets comments, and put pinned comments in cache */
export const setComments = (comments: StoreState['comments']): StoreAction<void> => dispatch => {
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
  const updatedComment = await api.getComment(id);
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
    await api.removeComment(id);
  } else {
    await api.removeMyComment(id);
  }
  const comments = getState().comments;
  dispatch(setComments(uRemoveComment(comments, id)));
};

/** fetches comments from server */
export const fetchComments = (sort: Sorting): StoreAction<Promise<Tree>> => async (dispatch, getState) => {
  const data = await api.getPostComments(sort);
  const hiddenUsersIds = Object.keys(getState().hiddenUsers);
  if (hiddenUsersIds.length > 0)
    data.comments = filterTree(data.comments, node => hiddenUsersIds.indexOf(node.comment.user.id) === -1);
  dispatch(setComments(data.comments));
  dispatch({
    type: POST_INFO_SET,
    info: data.info,
  });
  return data;
};

function getCommentHash(comment: Comment): string {
  return comment.id + (comment.edit ? comment.edit.time : '');
}

/** fetches comments from server */
export const fetchNewComments = (): StoreAction<Promise<void>> => async (dispatch, getState) => {
  const { comments, hiddenUsers } = getState();
  const data = await api.getPostCommentsList('+time');
  // get flat list of existing ids
  const currentCommentsIds = flattenTree(comments, c => c.id);
  // get flat list of existing hashes
  // the reason is that we need to distinguish new comments and edited comments
  // so we consider newComments further comments that has different hash instead of just id
  const currentCommentsHashes = flattenTree(comments, getCommentHash);
  const hiddenUsersIds = Object.keys(hiddenUsers);
  const newComments = data.comments.filter(
    c => currentCommentsHashes.indexOf(getCommentHash(c)) === -1 && hiddenUsersIds.indexOf(c.user.id) === -1
  );
  if (!newComments.length) return;

  let updatedComments = [...comments];
  for (const comment of newComments) {
    if (!comment.pid) {
      if (currentCommentsIds.indexOf(comment.id) === -1) {
        updatedComments.push({ comment });
      } else {
        updatedComments = uReplaceComment(updatedComments, comment);
      }
      continue;
    }
    if (currentCommentsIds.indexOf(comment.id) === -1) {
      updatedComments = uAddComment(updatedComments, { ...comment, new: true }, true);
    } else {
      updatedComments = uReplaceComment(updatedComments, comment);
    }
  }
  dispatch(setComments(updatedComments));
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

export const unwrapNewComments = (id: Comment['id']): StoreAction<void> => (dispatch, getState) => {
  const comments = mapTreeIfID(getState().comments, id, node => {
    if (!node.replies) return node;
    return { ...node, replies: mapTree(node.replies, c => ({ ...c, new: false })) };
  });
  dispatch(setComments(comments));
};
