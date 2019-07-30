import { Node, Comment } from '@app/common/types';
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
  ids: (Comment['id'])[];
  patch: Partial<Comment>;
}

export const COMMENT_MODE_SET = 'COMMENT_MODE/SET';

export interface COMMENT_MODE_SET_ACTION {
  type: typeof COMMENT_MODE_SET;
  mode: StoreState['activeComment'];
}

export type COMMENTS_ACTIONS =
  | COMMENTS_SET_ACTION
  | COMMENTS_APPEND_ACTION
  | COMMENTS_EDIT_ACTION
  | COMMENTS_PATCH_ACTION
  | COMMENT_MODE_SET_ACTION;
