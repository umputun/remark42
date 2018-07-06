/** @jsx h */
import { h, Component } from 'preact';

import api from 'common/api';

import Comment from 'components/comment';
import Preloader from 'components/preloader';

export default class UserInfo extends Component {
  constructor(props) {
    super(props);

    this.state = {
      comments: [],
      isLoading: true,
    };
  }

  componentWillMount() {
    const {
      user: { id },
    } = this.props;

    api
      .getUserComments({ user: id, limit: 10 })
      .then(({ comments = [] }) => this.setState({ comments }))
      .finally(() => this.setState({ isLoading: false }));
  }

  render(props, { comments, isLoading }) {
    const {
      user: { name, id },
      onClose,
    } = props;

    return (
      <div className={b('user-info', props)}>
        <p className="user-info__title">Last comments by {name}</p>
        <p className="user-info__id">{id}</p>

        {isLoading && <Preloader mix="user-info__preloader" />}

        {!isLoading && (
          <div>{comments.map(comment => <Comment data={comment} mods={{ level: 0, view: 'user' }} />)}</div>
        )}

        <div>
          <span className="user-info__close" onClick={onClose}>
            Close
          </span>
        </div>
      </div>
    );
  }
}
