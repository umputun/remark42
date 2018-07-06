import { h, Component } from 'preact';

import api from 'common/api';
import { A11yButton } from 'common/accessibility';
import { API_BASE, BASE_URL, COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants';
import { url } from 'common/settings';
import store from 'common/store';

import Input from 'components/input';
import UserInfo from 'components/user-info';

export default class Comment extends Component {
  constructor(props) {
    super(props);

    this.state = {
      isReplying: false,
      isEditing: false,
      isUserVerified: false,
      isUserInfoShown: false,
      editTimeLeft: null,
    };

    this.votingPromise = Promise.resolve();

    this.updateState(props);

    this.decreaseScore = this.decreaseScore.bind(this);
    this.increaseScore = this.increaseScore.bind(this);
    this.toggleEditing = this.toggleEditing.bind(this);
    this.toggleReplying = this.toggleReplying.bind(this);
    this.toggleCollapse = this.toggleCollapse.bind(this);
    this.toggleUserInfoVisibility = this.toggleUserInfoVisibility.bind(this);
    this.scrollToParent = this.scrollToParent.bind(this);
    this.onEdit = this.onEdit.bind(this);
    this.onReply = this.onReply.bind(this);
    this.onPinClick = this.onPinClick.bind(this);
    this.onUnpinClick = this.onUnpinClick.bind(this);
    this.onVerifyClick = this.onVerifyClick.bind(this);
    this.onUnverifyClick = this.onUnverifyClick.bind(this);
    this.onBlockClick = this.onBlockClick.bind(this);
    this.onUnblockClick = this.onUnblockClick.bind(this);
    this.onDeleteClick = this.onDeleteClick.bind(this);
  }

  componentWillReceiveProps(nextProps) {
    this.updateState(nextProps);
  }

  updateState(props) {
    const { data, data: { user: { block, id: commentUserId }, pin }, mods: { guest } = {} } = props;

    const votes = data && data.votes || [];
    const score = data && data.score || 0;

    if (this.editTimerInterval) {
      clearInterval(this.editTimerInterval);
      this.editTimerInterval = null;
    }

    if (guest) {
      this.setState({
        guest,
        score,
        deleted: data ? data.delete : false,
      });
    } else {
      const userId = store.get('user').id;

      this.setState({
        guest,
        score,
        pinned: !!pin,
        deleted: data ? data.delete : false,
        userBlocked: !!block,
        scoreIncreased: userId in votes && votes[userId],
        scoreDecreased: userId in votes && !votes[userId],
      });

      if (userId === commentUserId) {
        const editDuration = store.get('config') && store.get('config').edit_duration;
        const getEditTimeLeft = () => Math.floor(editDuration - (new Date() - new Date(data.time)) / 1000);

        if (getEditTimeLeft() > 0) {
          this.editTimerInterval = setInterval(() => {
            const editTimeLeft = getEditTimeLeft();

            if (editTimeLeft < 0) {
              clearInterval(this.editTimerInterval);
              this.editTimerInterval = null;
              this.setState({ editTimeLeft: null });
            } else {
              this.setState({ editTimeLeft });
            }
          }, 1000);
        }
      }
    }
  }

  toggleReplying() {
    const { isReplying } = this.state;
    const onPrevInputToggleCb = store.get('onPrevInputToggleCb');

    this.setState({ isEditing: false }, () => this.setState({ isReplying: !isReplying }));

    if (onPrevInputToggleCb) onPrevInputToggleCb();

    if (!isReplying) {
      store.set('onPrevInputToggleCb', () => this.setState({ isReplying: false }));
    } else {
      store.set('onPrevInputToggleCb', null);
    }
  }

  toggleEditing() {
    const { isEditing } = this.state;
    const onPrevInputToggleCb = store.get('onPrevInputToggleCb');

    this.setState({ isReplying: false }, () => this.setState({ isEditing: !isEditing }));

    if (onPrevInputToggleCb) onPrevInputToggleCb();

    if (!isEditing) {
      store.set('onPrevInputToggleCb', () => this.setState({ isEditing: false }));
    } else {
      store.set('onPrevInputToggleCb', null);
    }
  }

  toggleUserInfoVisibility() {
    this.setState({ isUserInfoShown: !this.state.isUserInfoShown });
  }

  onPinClick() {
    const { id } = this.props.data;

    if (confirm('Do you want to pin this comment?')) {
      this.setState({ pinned: true });

      api.pinComment({ id, url }).then(() => {
        api.getComment({ id }).then(comment => store.replaceComment(comment));
      });
    }
  }

  onUnpinClick() {
    const { id } = this.props.data;

    if (confirm('Do you want to unpin this comment?')) {
      this.setState({ pinned: false });

      api.unpinComment({ id, url }).then(() => {
        api.getComment({ id }).then(comment => store.replaceComment(comment));
      });
    }
  }

  onVerifyClick() {
    const { id, user: { id: userId } } = this.props.data;

    if (confirm('Do you want to verify this user?')) {
      this.setState({ isUserVerified: true });

      api.setVerifyStatus({ id: userId }).then(() => {
        api.getComment({ id }).then(comment => store.replaceComment(comment));
      });
    }
  }

  onUnverifyClick() {
    const { id, user: { id: userId } } = this.props.data;

    if (confirm('Do you want to unverify this user?')) {
      this.setState({ isUserVerified: false });

      api.removeVerifyStatus({ id: userId }).then(() => {
        api.getComment({ id }).then(comment => store.replaceComment(comment));
      });
    }
  }

  onBlockClick() {
    const { id, user: { id: userId } } = this.props.data;

    if (confirm('Do you want to block this user?')) {
      this.setState({ userBlocked: true });

      api.blockUser({ id: userId }).then(() => {
        api.getComment({ id }).then(comment => store.replaceComment(comment));
      });
    }
  }

  onUnblockClick() {
    const { id, user: { id: userId } } = this.props.data;

    if (confirm('Do you want to unblock this user?')) {
      this.setState({ userBlocked: false });

      api.unblockUser({ id: userId }).then(() => {
        api.getComment({ id }).then(comment => store.replaceComment(comment));
      });
    }
  }

  onDeleteClick() {
    const { id } = this.props.data;

    if (confirm('Do you want to delete this comment?')) {
      this.setState({
        deleted: true,
        isEditing: false,
        isReplying: false,
      });

      api.removeComment({ id }).then(() => {
        api.getComment({ id }).then(comment => store.replaceComment(comment));
      });
    }
  }

  increaseScore() {
    const { score, scoreIncreased, scoreDecreased } = this.state;
    const { id } = this.props.data;

    if (scoreIncreased) return;

    this.setState({
      scoreIncreased: !scoreDecreased,
      scoreDecreased: false,
      score: score + 1,
    });

    this.votingPromise = this.votingPromise
      .then(() => {
        return api.putCommentVote({ id, url, value: 1 })
          .then(() => {
            api.getComment({ id }).then(comment => store.replaceComment(comment));
          });
      });
  }

  decreaseScore() {
    const { score, scoreIncreased, scoreDecreased } = this.state;
    const { id } = this.props.data;

    if (scoreDecreased) return;

    this.setState({
      scoreDecreased: !scoreIncreased,
      scoreIncreased: false,
      score: score - 1,
    });

    this.votingPromise = this.votingPromise
      .then(() => {
        return api.putCommentVote({ id, url, value: -1 })
          .then(() => {
            api.getComment({ id }).then(comment => store.replaceComment(comment));
          });
      });
  }

  onReply(...rest) {
    this.props.onReply(...rest);

    this.setState({
      isReplying: false,
    });
  }

  onEdit(...rest) {
    this.props.onEdit(...rest);

    this.setState({
      isEditing: false,
    });
  }

  scrollToParent(e) {
    const { data: { pid } } = this.props;

    e.preventDefault();

    const parentCommentNode = document.getElementById(`${COMMENT_NODE_CLASSNAME_PREFIX}${pid}`);

    if (parentCommentNode) {
      parentCommentNode.scrollIntoView();
    }
  }

  toggleCollapse() {
    this.setState({
      isEditing: false,
      isReplying: false,
    });

    if (this.props.onCollapseToggle) {
      this.props.onCollapseToggle();
    }
  }

  render(props, {
    guest,
    userBlocked,
    pinned,
    score,
    scoreIncreased,
    scoreDecreased,
    deleted,
    isReplying,
    isEditing,
    isUserVerified,
    editTimeLeft,
    isUserInfoShown,
  }) {
    const { data, mods = {} } = props;
    const isAdmin = !guest && store.get('user').admin;
    const isGuest = guest || !Object.keys(store.get('user')).length;
    const isCurrentUser = (data.user && data.user.id) === (store.get('user') && store.get('user').id);
    const config = store.get('config') || {};
    const lowCommentScore = config.low_score;

    const o = {
      ...data,
      text:
        data.text.length
          ? (mods.view === 'preview' ? getTextSnippet(data.text) : data.text)
          : (
            userBlocked
              ? 'This user was blocked'
              : (
                deleted
                  ? 'This comment was deleted'
                  : data.text
              )
          ),
      time: formatTime(new Date(data.time)),
      orig: isEditing
        ? (
          data.orig && data.orig
            .replace(/&[#A-Za-z0-9]+;/gi, entity => {
              const span = document.createElement('span');
              span.innerHTML = entity;
              return span.innerText;
            })
        )
        : data.orig,
      score: {
        value: Math.abs(score),
        sign: score > 0 ? '+' : (score < 0 ? '−' : null),
        view: score > 0 ? 'positive' : (score < 0 ? 'negative' : null),
      },
      user: {
        ...data.user,
        picture: data.user.picture.indexOf(API_BASE) === 0 ? `${BASE_URL}${data.user.picture}` : data.user.picture,
        isDefaultPicture: !data.user.picture.length,
        verified: data.user.verified || isUserVerified,
      },
    };

    const defaultMods = {
      pinned,
      // TODO: we also have critical_score, so we need to collapse comments with it in future
      useless: userBlocked || deleted || (score <= lowCommentScore && !mods.pinned && !mods.disabled),
      // TODO: add default view mod or don't?
      view: o.user.admin ? 'admin' : null,
      replying: isReplying,
      editing: isEditing,
    };

    if (mods.view === 'preview') {
      return (
        <article className={b('comment', props, defaultMods)}>
          <div className="comment__body">
            <div className="comment__info">
              <a href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`} className="comment__username">{o.user.name}</a>
            </div>
            {' '}
            <div
              className={b('comment__text', { mix: 'raw-content' })}
              dangerouslySetInnerHTML={{ __html: o.text }}
            />
          </div>
        </article>
      );
    }

    return (
      <article className={b('comment', props, defaultMods)} id={mods.disabled ? null : `${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`}>
        <div className="comment__body">
          <div className="comment__info">
            {
              mods.view !== 'user' && (
                <img
                  className={b('comment__avatar', {}, { default: o.user.isDefaultPicture })}
                  src={o.user.isDefaultPicture ? require('./__avatar/comment__avatar.svg') : o.user.picture}
                  alt=""
                />
              )
            }

            {
              mods.view !== 'user' && (
                <A11yButton onClick={this.toggleUserInfoVisibility}>
                  <span
                    className="comment__username"
                    title={o.user.id}
                  >{o.user.name}</span>
                </A11yButton>
                
              )
            }

            {
              isAdmin && mods.view !== 'user' && (
                <A11yButton onClick={o.user.verified ? this.onUnverifyClick : this.onVerifyClick}>
                  <span
                    role="button"
                    tabIndex={0}
                    aria-label="Toggle verification"
                    title={o.user.verified ? 'Verified user' : 'Unverified user'}
                    className={b('comment__verification', {}, { active: o.user.verified, clickable: true })}
                  />
                </A11yButton>
              )
            }

            {
              !isAdmin && !!o.user.verified && mods.view !== 'user' && (
                <span
                  title="Verified user"
                  className={b('comment__verification', {}, { active: true })}
                />
              )
            }

            <a href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`} className="comment__time">{o.time}</a>

            {
              mods.level > 0 && mods.view !== 'user' && (
                <a
                  className="comment__link-to-parent"
                  href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.pid}`}
                  aria-label="Go to parent comment"
                  title="Go to parent comment"
                  onClick={this.scrollToParent}
                > </a>
              )
            }

            {
              isAdmin && userBlocked && mods.view !== 'user' && (
                <span className="comment__status">Blocked</span>
              )
            }

            {
              isAdmin && !userBlocked && deleted && (
                <span className="comment__status">Deleted</span>
              )
            }

            {
              !mods.disabled && mods.view !== 'user' && (
                <A11yButton onClick={this.toggleCollapse}>
                  <span
                    className={b('comment__action', {}, { type: 'collapse', selected: mods.collapsed })}
                  >{mods.collapsed ? '+' : '−'}</span>
                </A11yButton>
              )
            }

            <span className={b('comment__score', {}, { view: o.score.view })}>
              <A11yButton onClick={isGuest || isCurrentUser ? null : this.increaseScore}>
                <span
                  className={b('comment__vote', {}, { type: 'up', selected: scoreIncreased, disabled: isGuest || isCurrentUser })}
                  aria-disabled={isGuest || isCurrentUser}
                  title={isGuest ? 'Only authorized users are allowed to vote' : (isCurrentUser ? 'You can\'t vote for your own comment' : null)}
                >Vote up</span>
              </A11yButton>

              <span className="comment__score-value">
                {o.score.sign}{o.score.value}
              </span>

              <A11yButton onClick={isGuest || isCurrentUser ? null : this.decreaseScore}>
                <span
                  className={b('comment__vote', {}, { type: 'down', selected: scoreDecreased, disabled: isGuest || isCurrentUser })}
                  aria-disabled={isGuest || isCurrentUser ? 'true' : 'false'}
                  title={isGuest ? 'Only authorized users are allowed to vote' : (isCurrentUser ? 'You can\'t vote for your own comment' : null)}
                >Vote down</span>
              </A11yButton>
            </span>
          </div>

          <div
            className={b('comment__text', { mix: 'raw-content' })}
            dangerouslySetInnerHTML={{ __html: o.text }}
          />

          <div className="comment__actions">
            {
              !deleted && !mods.disabled && !isGuest && mods.view !== 'user' && (
                <A11yButton onClick={this.toggleReplying}>
                  <span
                    className="comment__action"
                  >{isReplying ? 'Cancel' : 'Reply'}</span>
                </A11yButton>
              )
            }

            {
              !deleted && !mods.disabled && !!o.orig && isCurrentUser && (!!editTimeLeft || isEditing) && mods.view !== 'user' && (
                <A11yButton onClick={this.toggleEditing}>
                  <span
                    className="comment__action comment__action_type_edit"
                  >{isEditing ? 'Cancel' : 'Edit'}{editTimeLeft && ` (${editTimeLeft})`}</span>
                </A11yButton>
              )
            }

            {
              !deleted && isAdmin && (
                <span className="comment__controls">
                  {
                    !pinned && (
                      <A11yButton onClick={this.onPinClick}>
                        <span
                          className="comment__control"
                        >Pin</span>
                      </A11yButton>
                    )
                  }

                  {
                    pinned && (
                      <A11yButton onClick={this.onUnpinClick}>
                        <span
                          className="comment__control"
                        >Unpin</span>
                      </A11yButton>
                    )
                  }

                  {
                    userBlocked && (
                      <A11yButton onClick={this.onUnblockClick}>
                        <span
                          className="comment__control"
                        >Unblock</span>
                      </A11yButton>
                    )
                  }

                  {
                    !userBlocked && (
                      <A11yButton onClick={this.onBlockClick}>
                        <span
                          className="comment__control"
                        >Block</span>
                      </A11yButton>
                    )
                  }

                  {
                    !deleted && (
                      <A11yButton onClick={this.onDeleteClick}>
                        <span
                          className="comment__control"
                        >Delete</span>
                      </A11yButton>
                    )
                  }
                </span>
              )
            }
          </div>
        </div>

        {
          isUserInfoShown && (
            <UserInfo mix="comment__user-info" user={o.user} onClose={this.toggleUserInfoVisibility}/>
          )
        }

        {
          isReplying && mods.view !== 'user' && (
            <Input
              mix="comment__input"
              onSubmit={this.onReply}
              onCancel={this.toggleReplying}
              pid={o.id}
            />
          )
        }

        {
          isEditing && mods.view !== 'user' && (
            <Input
              mix="comment__input"
              mods={{ mode: 'edit' }}
              onSubmit={this.onEdit}
              onCancel={this.toggleEditing}
              id={o.id}
              value={o.orig}
              errorMessage={!editTimeLeft && 'Editing time has expired.'}
            />
          )
        }
      </article>
    );
  }
}

function getTextSnippet(html) {
  const LENGTH = 100;
  const tmp = document.createElement('div');
  tmp.innerHTML = html.replace('</p><p>', ' ');

  const result = tmp.innerText || '';
  const snippet = result.substr(0, LENGTH);

  return (snippet.length === LENGTH && result.length !== LENGTH) ? `${snippet}...` : snippet;
}

function formatTime(time) {
  // 'ru-RU' adds a dot as a separator
  const date = time.toLocaleDateString(['ru-RU'], { day: '2-digit', month: '2-digit', year: '2-digit' });

  // do it manually because Intl API doesn't add leading zeros to hours; idk why
  const hours = `0${time.getHours()}`.slice(-2);
  const mins = `0${time.getMinutes()}`.slice(-2);

  return `${date} at ${hours}:${mins}`;
}
