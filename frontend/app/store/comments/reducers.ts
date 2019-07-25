import { Node, Comment, CommentMode } from '@app/common/types';

import {
  COMMENTS_SET,
  COMMENTS_SET_ACTION,
  PINNED_COMMENTS_SET_ACTION,
  PINNED_COMMENTS_SET,
  COMMENT_MODE_SET,
  COMMENT_MODE_SET_ACTION,
} from './types';

export const comments = (state: Node[] = [], action: COMMENTS_SET_ACTION): Node[] => {
  switch (action.type) {
    case COMMENTS_SET: {
      return action.comments;
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

export const pinnedComments = (state: Comment[] = [], action: PINNED_COMMENTS_SET_ACTION): Comment[] => {
  switch (action.type) {
    case PINNED_COMMENTS_SET: {
      return action.comments;
    }
    default:
      return state;
  }
};

export default { comments, activeComment, pinnedComments };
