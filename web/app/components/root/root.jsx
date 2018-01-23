import { h, Component } from 'preact';
import api from 'common/api';

import { url, id } from 'common/settings';

import Thread from 'components/thread';

export default class Root extends Component {
  componentDidMount() {
    // TODO: add preloader
    api.find({ url }).then(({ comments } = {}) => this.setState({ comments }));
  }

  render({}, { comments = [] }) {
    return (
      <div className="root" id={id}>
        {
          comments.map(thread => <Thread data={thread} mods={{ level: 0 }}/>)
        }
      </div>
    );
  }
}
