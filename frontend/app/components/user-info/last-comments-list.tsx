/** @jsx createElement */
import { createElement } from 'preact';

import { Comment as CommentType } from '@app/common/types';

import { Comment } from '@app/components/comment';
import { Preloader } from '@app/components/preloader';
import { useIntl } from 'react-intl';

const LastCommentsList = ({ comments, isLoading }: { comments: CommentType[]; isLoading: boolean }) => {
  const intl = useIntl();
  if (isLoading) {
    return <Preloader mix="user-info__preloader" />;
  }
  return (
    <div>
      {comments.map(comment => (
        <Comment
          CommentForm={null}
          data={comment}
          level={0}
          intl={intl}
          view="user"
          user={null}
          isCommentsDisabled={false}
          theme="light"
          post_info={null}
        />
      ))}
    </div>
  );
};

export default LastCommentsList;
