import { Node, Comment } from '@app/common/types';

import { StoreState } from '../index';
import {
  COMMENTS_SET,
  COMMENTS_SET_ACTION,
  PINNED_COMMENTS_SET_ACTION,
  PINNED_COMMENTS_SET,
  COMMENT_MODE_SET,
  COMMENT_MODE_SET_ACTION,
} from './types';

export const comments = (state: StoreState['comments'] = [], action: COMMENTS_SET_ACTION): Node[] => {
  switch (action.type) {
    case COMMENTS_SET: {
      return action.comments;
    }
    default:
      return state;
  }
};

export const activeComment = (
  state: StoreState['activeComment'] = null,
  action: COMMENT_MODE_SET_ACTION
): StoreState['activeComment'] => {
  switch (action.type) {
    case COMMENT_MODE_SET: {
      return action.mode;
    }
    default:
      return state;
  }
};

export const pinnedComments = (
  state: StoreState['pinnedComments'] = [],
  action: PINNED_COMMENTS_SET_ACTION
): Comment[] => {
  switch (action.type) {
    case PINNED_COMMENTS_SET: {
      return action.comments;
    }
    default:
      return state;
  }
};

export default { comments, activeComment, pinnedComments };
