import { Comment } from '@app/common/types';
import { htmlUnescape } from './htmlUnescape';

/**
 * comment received from api has name of user html escaped,
 * To render it well we must unescape this property
 */
export const unescapeComment = (comment: Comment): Comment => {
  return {
    ...comment,
    user: { ...comment.user, name: htmlUnescape(comment.user.name) },
  };
};
