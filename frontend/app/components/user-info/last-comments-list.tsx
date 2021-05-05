import { h, Fragment } from 'preact';
import { useIntl } from 'react-intl';

import { Comment as CommentType } from 'common/types';

import { Comment } from 'components/comment';
import { Preloader } from 'components/preloader';

type Props = {
  comments: CommentType[];
  isLoading: boolean;
};

export function LastCommentsList({ comments, isLoading }: Props) {
  const intl = useIntl();

  if (isLoading) {
    return <Preloader mix="user-info__preloader" />;
  }

  return (
    <>
      {comments.map((comment) => (
        <Comment
          key={comment.id}
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
}
