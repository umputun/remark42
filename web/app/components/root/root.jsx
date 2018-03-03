import { h, Component } from 'preact';
import api from 'common/api';

import { BASE_URL, NODE_ID } from 'common/constants';
import { url } from 'common/settings';
import store from 'common/store';

import AuthPanel from 'components/auth-panel';
import Comment from 'components/comment';
import Input from 'components/input';
import Preloader from 'components/preloader';
import Thread from 'components/thread';

export default class Root extends Component {
  constructor(props) {
    super(props);

    this.state = {
      loaded: false,
      user: {},
    };

    this.addComment = this.addComment.bind(this);
    this.onSignIn = this.onSignIn.bind(this);
    this.onSignOut = this.onSignOut.bind(this);
  }

  componentWillMount() {
    store.onUpdate('comments', comments => this.setState({ comments }));
  }

  componentDidMount() {
    api.getUser()
      .then(data => store.set('user', data))
      .catch(() => store.set('user', {}))
      .finally(() => {
        api.find({ url })
          .then(({ comments } = {}) => store.set('comments', comments))
          .finally(() => this.setState({
            loaded: true,
            user: store.get('user'),
          }));

        api.getConfig().then(config => {
          store.set('config', config);
          this.setState({ config });
        })
      });
  }

  onSignOut() {
    api.logOut().then(() => {
      store.set('user', {});
      this.setState({ user: {} });
    });
  }

  onSignIn(provider) {
    const newWindow = window.open(`${BASE_URL}/auth/${provider}/login?from=${encodeURIComponent(location.href)}`);

    let secondsPass = 0;
    const checkMsDelay = 200;
    const checkInterval = setInterval(() => {
      secondsPass += checkMsDelay;

      if (newWindow.location.origin === location.origin || secondsPass > 30000) {
        clearInterval(checkInterval);
        secondsPass = 0;
        newWindow.close();

        api.getUser()
          .then(user => {
            store.set('user', user);
            this.setState({ user });
          })
          .catch(() => {}) // TODO: we need to handle it and write error to user
      }
    }, checkMsDelay);
  }

  addComment(data) {
    store.addComment(data);
    this.setState({ comments: store.get('comments') });

    api.getComment({ id: data.id }).then(comment => {
      store.replaceComment(comment);
      this.setState({ comments: store.get('comments') });
    });
  }

  render({}, { config = {}, comments = [], user, loaded }) {
    if (!loaded) {
      return (
        <div id={NODE_ID}>
          <div className="root">
            <Preloader mix="root__preloader"/>
          </div>
        </div>
      );
    }

    // TODO: i think we should do it on backend
    const pinnedComments = store.getPinnedComments();

    return (
      <div id={NODE_ID}>
        <div className="root">
          <AuthPanel
            mix="root__auth-panel"
            user={user}
            providers={config.auth_providers}
            onSignIn={this.onSignIn}
            onSignOut={this.onSignOut}
          />

          <Input
            mix="root__input"
            onSubmit={this.addComment}
          />

          {
            !!pinnedComments.length && (
              <div className="root__pinned-comments">
                {
                  pinnedComments.map(comment => (
                    <Comment
                      data={comment}
                      mods={{ level: 0, disabled: true }}
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
