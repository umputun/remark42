import { h, Component } from 'preact';
import api from 'common/api';

import { url, id } from 'common/settings';

import Input from 'components/input';
import Thread from 'components/thread';

export default class Root extends Component {
  constructor(props) {
    super(props);

    this.addThread = this.addThread.bind(this);
  }

  componentDidMount() {
    // TODO: add preloader
    // there must be request for checking auth
    api.find({ url }).then(({ comments } = {}) => this.setState({ comments }));
  }

  addThread() {
    // nothing here yet
  }

  render({}, { comments = [], user = {} }) {
    return (
      <div>
        <div className="root" id={id}>
          <Input mix="root__input" onSubmit={this.addThread}/>

          {
            comments.map(thread => <Thread data={thread} mods={{ level: 0 }} mix="root__thread"/>)
          }
        </div>
      </div>
    );
  }
}
