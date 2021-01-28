import { Node, Comment, CommentMode, Sorting } from 'common/types';
import { combineReducers } from 'redux';

import {
  COMMENTS_SET,
  COMMENTS_SET_ACTION,
  COMMENT_MODE_SET,
  COMMENT_MODE_SET_ACTION,
  COMMENTS_APPEND_ACTION,
  COMMENTS_APPEND,
  COMMENTS_EDIT_ACTION,
  COMMENTS_EDIT,
  COMMENTS_PATCH,
  COMMENTS_PATCH_ACTION,
  COMMENTS_SET_SORT,
  COMMENTS_SET_SORT_ACTION,
  COMMENTS_REQUEST_FETCHING,
  COMMENTS_REQUEST_SUCCESS,
  COMMENTS_REQUEST_FAILURE,
  COMMENTS_REQUEST_ACTIONS,
} from './types';
import { getPinnedComments, getInitialSort } from './utils';
import { cmpRef } from 'utils/cmpRef';

export const topComments = (
  state: Comment['id'][] = [],
  action: COMMENTS_SET_ACTION | COMMENTS_APPEND_ACTION
): Comment['id'][] => {
  switch (action.type) {
    case COMMENTS_SET: {
      return cmpRef(
        state,
        action.comments.map((x) => x.comment.id)
      );
    }
    case COMMENTS_APPEND: {
      if (action.comment.pid) return state;
      return [action.comment.id, ...state];
    }
    default:
      return state;
  }
};

const reduceChildIds = (c: Record<Comment['id'], Comment['id'][]>, x: Node): Record<Comment['id'], Comment['id'][]> => {
  if (!x.replies) return c;
  if (!c[x.comment.id]) {
    c[x.comment.id] = [];
  }
  for (const reply of x.replies) {
    c[x.comment.id].push(reply.comment.id);
    if (reply.replies) {
      reduceChildIds(c, reply);
    }
  }

  return c;
};

export const childComments = (
  state: Record<Comment['id'], Comment['id'][]> = {},
  action: COMMENTS_SET_ACTION | COMMENTS_APPEND_ACTION
): Record<Comment['id'], Comment['id'][]> => {
  switch (action.type) {
    case COMMENTS_SET: {
      return action.comments.reduce<Record<Comment['id'], Comment['id'][]>>(reduceChildIds, {});
    }
    case COMMENTS_APPEND: {
      if (!action.comment.pid) return state;
      return { ...state, [action.comment.pid]: [action.comment.id, ...(state[action.comment.pid] || [])] };
    }
    default:
      return state;
  }
};

const cmpComment = (a: Comment | undefined, b: Comment): Comment => {
  if (!a) return b;
  if (a.id !== b.id) return b;
  if (!a.edit) {
    if (!b.edit) return a;
    return b;
  }
  if (!b.edit) return b;
  if (a.edit.time !== b.edit.time) return b;
  return a;
};

const reduceComments = (c: Record<Comment['id'], Comment>, x: Node): Record<Comment['id'], Comment> => {
  c[x.comment.id] = cmpComment(c[x.comment.id], x.comment);
  if (x.replies) {
    x.replies.reduce(reduceComments, c);
  }
  return c;
};

export const allComments = (
  state: Record<Comment['id'], Comment> = {},
  action: COMMENTS_SET_ACTION | COMMENTS_APPEND_ACTION | COMMENTS_EDIT_ACTION | COMMENTS_PATCH_ACTION
): Record<Comment['id'], Comment> => {
  switch (action.type) {
    case COMMENTS_SET: {
      return action.comments.reduce<Record<Comment['id'], Comment>>(reduceComments, { ...state });
    }
    case COMMENTS_APPEND:
    case COMMENTS_EDIT: {
      return { ...state, [action.comment.id]: action.comment };
    }
    case COMMENTS_PATCH: {
      let newState = state;
      let changed = false;
      const editObject = { summary: '', time: new Date().toISOString() };
      for (const id of action.ids) {
        if (!Object.prototype.hasOwnProperty.call(state, id)) continue;
        if (!changed) {
          changed = true;
          newState = { ...newState };
        }
        newState[id] = { ...newState[id], edit: editObject, ...action.patch };
      }
      return newState;
    }
    default:
      return state;
  }
};

export type ActiveCommentState = null | { id: Comment['id']; state: CommentMode };

export const activeComment = (
  state: ActiveCommentState = null,
  action: COMMENT_MODE_SET_ACTION
): ActiveCommentState => {
  switch (action.type) {
    case COMMENT_MODE_SET: {
      return action.mode;
    }
    default:
      return state;
  }
};

export const pinnedComments = (
  state: Comment['id'][] = [],
  action: COMMENTS_SET_ACTION | COMMENTS_EDIT_ACTION | COMMENTS_PATCH_ACTION
): Comment['id'][] => {
  switch (action.type) {
    case COMMENTS_SET: {
      return getPinnedComments(action.comments).map((x) => x.id);
    }
    case COMMENTS_EDIT: {
      const index = state.indexOf(action.comment.id);
      if (!action.comment.pin) {
        if (index === -1) return state;
        const newState = [...state];
        newState.splice(index, 1);
        return newState;
      }
      if (index !== -1) return state;
      return [...state, action.comment.id];
    }
    case COMMENTS_PATCH: {
      if (!Object.prototype.hasOwnProperty.call(action.patch, 'pin')) return state;
      if (!action.patch.pin) {
        return state.filter((x) => action.ids.indexOf(x) === -1);
      }
      return [...state, ...action.ids].reduce<Comment['id'][]>((c, x) => {
        if (c.indexOf(x) === -1) {
          c.push(x);
        }
        return c;
      }, []);
    }

    default:
      return state;
  }
};

function isFetching(state = false, action: COMMENTS_REQUEST_ACTIONS): boolean {
  switch (action.type) {
    case COMMENTS_REQUEST_FETCHING:
      return true;
    case COMMENTS_REQUEST_SUCCESS:
    case COMMENTS_REQUEST_FAILURE:
      return false;
    default:
      return state;
  }
}

function sort(state: Sorting = getInitialSort(), action: COMMENTS_SET_SORT_ACTION): Sorting {
  switch (action.type) {
    case COMMENTS_SET_SORT:
      return action.payload;
    default:
      return state;
  }
}

export default combineReducers({
  sort,
  isFetching,
  topComments,
  childComments,
  allComments,
  activeComment,
  pinnedComments,
});
