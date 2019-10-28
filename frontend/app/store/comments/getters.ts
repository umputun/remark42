import { Comment, CommentMode } from '@app/common/types';
import { StoreState } from '../index';

export const getCommentMode = (id: Comment['id']) => (state: StoreState): CommentMode => {
  if (state.activeComment === null) {
    return CommentMode.None;
  }
  if (state.activeComment.id !== id) {
    return CommentMode.None;
  }
  return state.activeComment.state;
};
