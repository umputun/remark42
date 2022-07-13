import { h } from 'preact';
import { useIntl } from 'react-intl';

import type { Comment as CommentType } from 'common/types';
import { Comment } from 'components/comment';

type Props = {
  comments: CommentType[];
};

export function ListComments({ comments = [] }: Props) {
  const intl = useIntl();

  return (
    <div className="comments-list">
      {comments.map((comment) => (
        <Comment
          intl={intl}
          key={comment.id}
          data={comment}
          level={0}
          view="preview"
          mix="list-comments__item"
          user={null}
          theme="light"
          isCommentsDisabled={false}
        />
      ))}
    </div>
  );
}
