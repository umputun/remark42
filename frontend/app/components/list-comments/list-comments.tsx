import { h } from 'preact';
import { useIntl } from 'react-intl';
import clsx from 'clsx';

import type { Comment as CommentType } from 'common/types';
import { Comment } from 'components/comment';

import styles from './list-comments.module.css';

type Props = {
  comments: CommentType[];
};

export function ListComments({ comments = [] }: Props) {
  const intl = useIntl();

  return (
    <div className={clsx('comments-list', styles.root)}>
      {comments.map((comment) => (
        <Comment
          intl={intl}
          key={comment.id}
          CommentForm={null}
          data={comment}
          level={0}
          view="preview"
          mix="list-comments__item"
          user={null}
          theme="light"
          isCommentsDisabled={false}
          post_info={null}
        />
      ))}
    </div>
  );
}
