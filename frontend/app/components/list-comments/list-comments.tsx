/** @jsx createElement */
import { createElement } from 'preact';

import { NODE_ID } from '@app/common/constants';
import { Comment as CommentType } from '@app/common/types';

import { Comment } from '@app/components/comment';
import { useIntl } from 'react-intl';

interface Props {
  comments: CommentType[];
}

export const ListComments = ({ comments = [] }: Props) => {
  const intl = useIntl();
  return (
    <div id={NODE_ID}>
      <div className="list-comments">
        {comments.map(comment => (
          <Comment
            CommentForm={null}
            data={comment}
            intl={intl}
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
