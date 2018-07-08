/** @jsx h */
import { h } from 'preact';

import Comment from 'components/comment';
import Preloader from 'components/preloader';

const LastCommentsList = ({ comments, isLoading }) => {
  if (isLoading) {
    return <Preloader mix="user-info__preloader" />;
  }

  return <div>{comments.map(comment => <Comment data={comment} mods={{ level: 0, view: 'user' }} />)}</div>;
};

export default LastCommentsList;
