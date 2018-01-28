import { h, Component } from 'preact';
import api from 'common/api';

import { url, id } from 'common/settings';

import Input from 'components/input';
import Thread from 'components/thread';

export default class Root extends Component {
  componentDidMount() {
    // TODO: add preloader
    api.find({ url }).then(({ comments } = {}) => this.setState({ comments }));
  }

  render({}, { comments = [] }) {
    return (
      <div>
        <div className="root" id={id}>
          <Input mix="root__input"/>

          {
            comments.map(thread => <Thread data={thread} mods={{ level: 0 }} mix="root__thread"/>)
          }
        </div>
      </div>
    );
  }
}
