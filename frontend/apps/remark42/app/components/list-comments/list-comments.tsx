import { h } from 'preact';
import { useIntl } from 'react-intl';

import type { Comment as CommentType } from 'common/types';
import { Comment } from 'components/comment';

import styles from './list-comments.module.css';

type Props = {
  comments: CommentType[];
};

export function ListComments({ comments = [] }: Props) {
  const intl = useIntl();

  return (
    <div>
      {comments.map((comment) => (
        <Comment
          intl={intl}
          key={comment.id}
          data={comment}
          level={0}
          view="preview"
          mix={styles.item}
          user={null}
          theme="light"
          isCommentsDisabled={false}
        />
      ))}
    </div>
  );
}
