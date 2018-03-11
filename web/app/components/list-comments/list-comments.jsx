import { h, Component } from 'preact';

import { NODE_ID } from 'common/constants';

import Comment from 'components/comment';

export default class ListComments extends Component {
  render({ comments = [] }, state) {
    return (
      <div id={NODE_ID}>
        <div className="list-comments">
          {
            comments.map(comment => (
              <Comment
                data={comment}
                mods={{ level: 0, guest: true }}
                mix="list-comments__item"
              />
            ))
          }
        </div>
      </div>
    );
  }
}
