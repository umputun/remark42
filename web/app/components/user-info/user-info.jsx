/** @jsx h */
import { h, Component } from 'preact';

import api from 'common/api';
import { getHandleClickProps } from 'common/accessibility';

import LastCommentsList from './last-comments-list';
import Avatar from 'components/avatar-icon';

class UserInfo extends Component {
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
      user: { name, id, isDefaultPicture, picture },
      onClose,
    } = props;

    return (
      <div className={b('user-info', props)}>
        <Avatar className="user-info__avatar" picture={isDefaultPicture ? null : picture} />
        <p className="user-info__title">Last comments by {name}</p>
        <p className="user-info__id">{id}</p>

        <LastCommentsList isLoading={isLoading} comments={comments} />

        <span {...getHandleClickProps(onClose)} className="user-info__close">
          Close &#10006;
        </span>
      </div>
    );
  }
}

export default UserInfo;
