import { h, Component } from 'preact';
import api from 'common/api';

import { url, id } from 'common/settings';
import store from 'common/store';

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
      .then(data => store.set({ user: data }))
      .catch(() => store.set({ user: {} }))
      .finally(() => {
        api.find({ url })
          .then(({ comments } = {}) => this.setState({ comments }))
          .finally(() => this.setState({ loaded: true }));
      });
  }

  addComment({ text, id, pid }) {
    const { comments } = this.state;
    const newComment = {
      comment: {
        id,
        text,
        user: store.get('user'),
        time: new Date(),
        ...(pid ? { pid } : {}),
      },
    };

    const newComments = pid
      ? Root.pasteReply({ comments, newComment })
      : [newComment].concat(comments);

    this.setState({ comments: newComments });

    api.getComment({ id }).then(comment => {
      this.setState({
        comments: Root.replaceComment({ comments: newComments, newComment: comment }),
      });
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

    return (
      <div id={id}>
        <div className="root root__loading" id={id}>
          <Input mix="root__input" onSubmit={this.addComment}/>

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

  static pasteReply({ comments, newComment }) {
    let again = true;

    const concatReply = (root, reply) => {
      root.replies = root.replies || [];
      root.replies = [reply].concat(root.replies);

      again = false;

      return root;
    }

    const paste = (root, commentObj) => {
      if (!again) return root;

      if (root.comment.id === commentObj.comment.pid) {
        return concatReply(root, commentObj);
      }

      if (root.replies) {
        root.replies = root.replies.map(reply => {
          if (reply.comment.id === commentObj.comment.pid) {
            return concatReply(reply, commentObj);
          } else {
            return paste(reply, commentObj);
          }
        })
      }

      return root;
    };

    return comments.map(thread => paste(thread, newComment));
  }

  static replaceComment({ comments, newComment }) {
    let again = true;

    const replace = (thread, comment) => {
      if (!again) return thread;

      if (thread.comment.id === comment.id) {
        thread.comment = comment;
        again = false;
        return thread;
      }

      thread.replies = thread.replies.map(reply => replace(reply, comment));

      return thread;
    };

    return comments.map(thread => replace(thread, newComment));
  }
}
