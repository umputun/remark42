import { h, Component } from 'preact';
import api from 'common/api';

import { url, id } from 'common/settings';
import store from 'common/store';

import Input from 'components/input';
import Thread from 'components/thread';

export default class Root extends Component {
  constructor(props) {
    super(props);

    this.addThread = this.addThread.bind(this);
  }

  componentDidMount() {
    api.user()
      .then(data => store.set({ user: data }))
      .catch(() => store.set({ user: {} }))
      .finally(() => {
        api.find({ url })
          .then(({ comments } = {}) => this.setState({ comments }))
          .finally(() => this.setState({ loaded: true }));
      });
  }

  addThread({ text, id }) {
    const { comments } = this.state;

    const newComments = [{
      comment: {
        id,
        text,
        user: store.get('user'),
        time: new Date(),
      },
    }].concat(comments);

    this.setState({ comments: newComments });
  }

  render({}, { comments = [], user = {}, loaded = false }) {
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
          <Input mix="root__input" onSubmit={this.addThread}/>

          {
            comments.map(thread => <Thread data={thread} mods={{ level: 0 }} mix="root__thread"/>)
          }
        </div>
      </div>
    );
  }
}
