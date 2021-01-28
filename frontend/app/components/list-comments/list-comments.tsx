import { h, FunctionComponent } from 'preact';
import { useIntl } from 'react-intl';
import classnames from 'classnames';

import type { Comment as CommentType } from 'common/types';
import Comment from 'components/comment';

import styles from './list-comments.module.css';

export type ListCommentsProps = {
  comments: CommentType[];
};

const ListComments: FunctionComponent<ListCommentsProps> = ({ comments = [] }) => {
  const intl = useIntl();

  return (
    <div className={classnames('comments-list', styles.root)}>
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
};

export default ListComments;
