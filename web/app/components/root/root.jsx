import { h, Component } from 'preact';
import api from 'common/api';

import { url, id } from 'common/settings';
import store from 'common/store';

import Comment from 'components/comment';
import Input from 'components/input';
import Thread from 'components/thread';

export default class Root extends Component {
  constructor(props) {
    super(props);

    this.state = {
      loaded: false,
    };

    this.addComment = this.addComment.bind(this);
  }

  componentDidMount() {
    api.getUser()
      .then(data => store.set('user', data))
      .catch(() => store.set('user', {}))
      .finally(() => {
        api.find({ url })
          .then(({ comments } = {}) => {
            store.set('comments', comments);
            this.setState({ comments });
          })
          .finally(() => this.setState({ loaded: true }));
      });
  }

  addComment(data) {
    store.addComment(data);
    this.setState({ comments: store.get('comments') });

    api.getComment({ id: data.id }).then(comment => {
      store.replaceComment(comment);
      this.setState({ comments: store.get('comments') });
    });
  }

  render({}, { comments = [], user = {}, loaded }) {
    if (!loaded) {
      return (
        <div id={id}>
          <div className="root root_loading"/>
        </div>
      );
    }

    // TODO: i think we should do it on backend
    const pinnedComments = store.getPinnedComments();

    return (
      <div id={id}>
        <div className="root root__loading" id={id}>
          <Input mix="root__input" onSubmit={this.addComment}/>

          {
            !!pinnedComments.length && (
              <div className="root__pinned-comments">
                {
                  pinnedComments.map(comment => (
                    <Comment
                      data={comment}
                      mods={{ level: 0 }}
                      mix="root__pinned-comment"
                    />
                  ))
                }
              </div>
            )
          }

          {
            comments.map(thread => (
              <Thread
                mix="root__thread"
                mods={{ level: 0 }}
                data={thread}
                onReply={this.addComment}
              />
            ))
          }
        </div>
      </div>
    );
  }
}
