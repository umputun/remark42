import type { Comment } from 'common/types';
import { CommentMode } from 'common/types';
import type { StoreState } from '../index';

export const getCommentMode =
  (id: Comment['id']) =>
  (state: StoreState): CommentMode => {
    if (state.comments.activeComment === null || state.comments.activeComment.id !== id) {
      return CommentMode.None;
    }

    return state.comments.activeComment.state;
  };
