/** @jsx h */
import { h } from 'preact';

import { NODE_ID } from 'common/constants';

import Comment from 'components/comment';

const ListComments = ({ comments = [] }) => (
  <div id={NODE_ID}>
    <div className="list-comments">
      {comments.map(comment => (
        <Comment data={comment} mods={{ level: 0, guest: true, view: 'preview' }} mix="list-comments__item" />
      ))}
    </div>
  </div>
);

export default ListComments;
