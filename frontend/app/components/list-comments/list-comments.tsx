/** @jsx createElement */
import { createElement, FunctionComponent } from 'preact';

import { NODE_ID } from '@app/common/constants';
import { Comment as CommentType } from '@app/common/types';
import { Comment } from '@app/components/comment';

export const ListComments: FunctionComponent<{ comments: CommentType[] }> = ({ comments = [] }) => (
  <div id={NODE_ID}>
    <div className="list-comments">
      {comments.map(comment => (
        <Comment
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
