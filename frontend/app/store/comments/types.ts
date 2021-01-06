import { Node, Comment, Sorting } from 'common/types';
import { StoreState } from '../index';

export const COMMENTS_SET = 'COMMENTS/SET';

export interface COMMENTS_SET_ACTION {
  type: typeof COMMENTS_SET;
  comments: Node[];
}

export const COMMENTS_APPEND = 'COMMENTS/APPEND';

export interface COMMENTS_APPEND_ACTION {
  type: typeof COMMENTS_APPEND;
  comment: Comment;
}

export const COMMENTS_EDIT = 'COMMENTS/EDIT';

export interface COMMENTS_EDIT_ACTION {
  type: typeof COMMENTS_EDIT;
  comment: Comment;
}

export const COMMENTS_PATCH = 'COMMENTS/PATCH';

export interface COMMENTS_PATCH_ACTION {
  type: typeof COMMENTS_PATCH;
  ids: Comment['id'][];
  patch: Partial<Comment>;
}

export const COMMENT_MODE_SET = 'COMMENT_MODE/SET';

export interface COMMENT_MODE_SET_ACTION {
  type: typeof COMMENT_MODE_SET;
  mode: StoreState['comments']['activeComment'];
}

export const COMMENTS_REQUEST_FETCHING = 'COMMENTS/FETCHING';
export const COMMENTS_REQUEST_SUCCESS = 'COMMENTS/FETCHING_SUCCESS';
export const COMMENTS_REQUEST_FAILURE = 'COMMENTS/FETCHING_FAILURE';

export type COMMENTS_REQUEST_ACTIONS_TYPE =
  | typeof COMMENTS_REQUEST_FETCHING
  | typeof COMMENTS_REQUEST_SUCCESS
  | typeof COMMENTS_REQUEST_FAILURE;

export interface COMMENTS_REQUEST_ACTIONS {
  type: COMMENTS_REQUEST_ACTIONS_TYPE;
}

export const COMMENTS_SET_SORT = 'COMMENTS/SET_SORT';

export interface COMMENTS_SET_SORT_ACTION {
  type: typeof COMMENTS_SET_SORT;
  payload: Sorting;
}

export type COMMENTS_ACTIONS =
  | COMMENTS_SET_ACTION
  | COMMENTS_APPEND_ACTION
  | COMMENTS_EDIT_ACTION
  | COMMENTS_PATCH_ACTION
  | COMMENT_MODE_SET_ACTION
  | COMMENTS_SET_SORT_ACTION
  | COMMENTS_REQUEST_ACTIONS;
