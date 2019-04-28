import { Comment, CommentMode } from '@app/common/types';
import { StoreState } from '../index';

export const getCommentMode = (state: StoreState, id: Comment['id']): CommentMode => {
  if (state.activeComment === null) {
    return CommentMode.None;
  }
  if (state.activeComment.id !== id) {
    return CommentMode.None;
  }
  return state.activeComment.state;
};
