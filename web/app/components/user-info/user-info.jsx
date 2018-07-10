/** @jsx h */
import { h, Component } from 'preact';
import { connect } from 'preact-redux';

import api from 'common/api';
import { getHandleClickProps } from 'common/accessibility';

import LastCommentsList from './last-comments-list';
import Avatar from 'components/avatar-icon';
import { fetchComments, completeFetchComments } from './user-info.actions';
import { getUserComments, getIsLoadingUserComments } from './user-info.getters';

class UserInfo extends Component {
  componentWillMount() {
    const {
      user: { id },
      comments,
      isLoading,
      fetchComments,
      completeFetchComments,
    } = this.props;

    if (!comments && !isLoading) {
      fetchComments(id);

      api
        .getUserComments({ user: id, limit: 10 })
        .then(({ comments }) => completeFetchComments(id, comments))
        .catch(() => completeFetchComments(id, []));
    }
  }

  render(props) {
    const {
      user: { name, id, isDefaultPicture, picture },
      comments = [],
      isLoading,
      onClose,
    } = props;

    return (
      <div className={b('user-info', props)}>
        <Avatar className="user-info__avatar" picture={isDefaultPicture ? null : picture} />
        <p className="user-info__title">Last comments by {name}</p>
        <p className="user-info__id">{id}</p>

        {!!comments && <LastCommentsList isLoading={isLoading} comments={comments} />}

        <span {...getHandleClickProps(onClose)} className="user-info__close">
          Close &#10006;
        </span>
      </div>
    );
  }
}

export default connect(
  (state, props) => ({
    comments: getUserComments(state, props.user.id),
    isLoading: getIsLoadingUserComments(state, props.user.id),
  }),
  { fetchComments, completeFetchComments }
)(UserInfo);
