import { Node, Comment } from '@app/common/types';
import { StoreState } from '../index';

export const COMMENTS_SET = 'COMMENTS/SET';

export interface COMMENTS_SET_ACTION {
  type: typeof COMMENTS_SET;
  comments: Node[];
}

export const COMMENTS_SET_MODE = 'COMMENTS/SET_MODE';

export interface COMMENTS_SET_MODE_ACTION {
  type: typeof COMMENTS_SET_MODE;
  mode: StoreState['activeComment'];
}

export const COMMENTS_APPEND = 'COMMENTS/APPEND';

export interface COMMENTS_APPEND_ACTION {
  type: typeof COMMENTS_APPEND;
  comments: Node;
}

export const PINNED_COMMENTS_SET = 'PINNED_COMMENTS/SET';

export interface PINNED_COMMENTS_SET_ACTION {
  type: typeof PINNED_COMMENTS_SET;
  comments: Comment[];
}

export const COMMENTS_FETCH_TREE = 'COMMENTS/FETCH_TREE';

export interface COMMENTS_FETCH_TREE_ACTION {
  type: typeof COMMENTS_FETCH_TREE;
  comments: Node[];
}

export const COMMENTS_SET_READONLY = 'COMMENTS/SET_READONLY';

export interface COMMENTS_SET_READONLY_ACTION {
  type: typeof COMMENTS_SET_READONLY;
  readonly: boolean;
}

export type COMMENTS_ACTIONS =
  | COMMENTS_SET_ACTION
  | COMMENTS_SET_MODE_ACTION
  | COMMENTS_APPEND_ACTION
  | PINNED_COMMENTS_SET_ACTION
  | COMMENTS_FETCH_TREE_ACTION
  | COMMENTS_SET_READONLY_ACTION;
