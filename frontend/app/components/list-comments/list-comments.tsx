import { h, FunctionComponent } from 'preact';
import { useIntl } from 'react-intl';

import type { Comment as CommentType } from 'common/types';
import { NODE_ID } from 'common/constants';
import Comment from 'components/comment';

export type ListCommentsProps = {
  comments: CommentType[];
};

const ListComments: FunctionComponent<ListCommentsProps> = ({ comments = [] }) => {
  const intl = useIntl();

  return (
    <div id={NODE_ID}>
      <div className="list-comments">
        {comments.map(comment => (
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
    </div>
  );
};

export default ListComments;
