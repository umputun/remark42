import { h, Fragment } from 'preact';

import { Comment as CommentType } from 'common/types';

import Comment from 'components/comment';
import Preloader from 'components/preloader';
import { useIntl } from 'react-intl';

const LastCommentsList = ({ comments, isLoading }: { comments: CommentType[]; isLoading: boolean }) => {
  const intl = useIntl();

  if (isLoading) {
    return <Preloader mix="user-info__preloader" />;
  }

  return (
    <>
      {comments.map((comment) => (
        <Comment
          CommentForm={null}
          intl={intl}
          data={comment}
          level={0}
          view="user"
          user={null}
          isCommentsDisabled={false}
          theme="light"
          post_info={null}
        />
      ))}
    </>
  );
};

export default LastCommentsList;
