import { Node } from '@app/common/types';
import { StoreState } from '../index';

export const COMMENTS_SET = 'COMMENTS/SET';

export interface COMMENTS_SET_ACTION {
  type: typeof COMMENTS_SET;
  comments: StoreState['comments'];
}

export const COMMENTS_APPEND = 'COMMENTS/APPEND';

export interface COMMENTS_APPEND_ACTION {
  type: typeof COMMENTS_APPEND;
  comments: Node;
}

export const PINNED_COMMENTS_SET = 'PINNED_COMMENTS/SET';

export interface PINNED_COMMENTS_SET_ACTION {
  type: typeof PINNED_COMMENTS_SET;
  comments: StoreState['pinnedComments'];
}

export const COMMENT_MODE_SET = 'COMMENT_MODE/SET';

export interface COMMENT_MODE_SET_ACTION {
  type: typeof COMMENT_MODE_SET;
  mode: StoreState['activeComment'];
}

export type COMMENTS_ACTIONS =
  | COMMENTS_SET_ACTION
  | COMMENTS_APPEND_ACTION
  | PINNED_COMMENTS_SET_ACTION
  | COMMENT_MODE_SET_ACTION;
