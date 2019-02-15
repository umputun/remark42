/** @jsx h */
import { h } from 'preact';

import { Comment as CommentType, PostInfo } from '@app/common/types';

import { Comment } from '../comment';
import Preloader from '../preloader';

const LastCommentsList = ({ comments, isLoading }: { comments: CommentType[]; isLoading: boolean }) => {
  if (isLoading) {
    return <Preloader mix="user-info__preloader" />;
  }

  return (
    <div>
      {comments.map(comment => (
        <Comment
          data={comment}
          level={0}
          view="user"
          user={null}
          isCommentsDisabled={false}
          theme="light"
          post_info={{} as PostInfo}
        />
      ))}
    </div>
  );
};

export default LastCommentsList;
