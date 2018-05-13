import { h, Component } from 'preact';

import api from 'common/api';
import { API_BASE, BASE_URL, COMMENT_NODE_CLASSNAME_PREFIX, USELESS_COMMENT_SCORE } from 'common/constants';
import { url } from 'common/settings';
import store from 'common/store';

import Input from 'components/input';

export default class Comment extends Component {
  constructor(props) {
    super(props);

    this.state = {
      isInputVisible: false,
      isUserIdVisible: false,
    };

    this.updateState(props);

    this.decreaseScore = this.decreaseScore.bind(this);
    this.increaseScore = this.increaseScore.bind(this);
    this.toggleInputVisibility = this.toggleInputVisibility.bind(this);
    this.toggleCollapse = this.toggleCollapse.bind(this);
    this.toggleUserIdVisibility = this.toggleUserIdVisibility.bind(this);
    this.scrollToParent = this.scrollToParent.bind(this);
    this.onReply = this.onReply.bind(this);
    this.onPinClick = this.onPinClick.bind(this);
    this.onUnpinClick = this.onUnpinClick.bind(this);
    this.onBlockClick = this.onBlockClick.bind(this);
    this.onUnblockClick = this.onUnblockClick.bind(this);
    this.onDeleteClick = this.onDeleteClick.bind(this);
  }

  componentWillReceiveProps(nextProps) {
    this.updateState(nextProps);
  }

  updateState(props) {
    const { data, data: { user: { block }, pin }, mods: { guest } = {} } = props;

    const votes = data && data.votes || [];
    const score = data && data.score || 0;

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
    }
  }

  toggleInputVisibility() {
    const { isInputVisible } = this.state;
    const onPrevReplyCb = store.get('onPrevReplyCb');

    this.setState({ isInputVisible: !isInputVisible });

    if (onPrevReplyCb) onPrevReplyCb();

    if (!isInputVisible) {
      store.set('onPrevReplyCb', () => this.setState({ isInputVisible: false }));
    } else {
      store.set('onPrevReplyCb', null);
    }
  }

  toggleUserIdVisibility() {
    this.setState({ isUserIdVisible: !this.state.isUserIdVisible });
  }

  onPinClick() {
    const { id } = this.props.data;

    if (confirm('Do you want to pin this comment?')) {
      this.setState({ pinned: true });

      api.pin({ id, url }).then(() => {
        api.getComment({ id }).then(comment => store.replaceComment(comment));
      });
    }
  }

  onUnpinClick() {
    const { id } = this.props.data;

    if (confirm('Do you want to unpin this comment?')) {
      this.setState({ pinned: false });

      api.unpin({ id, url }).then(() => {
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
      this.setState({ deleted: true });

      api.remove({ id }).then(() => {
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

    api.vote({ id, url, value: 1 }).then(() => {
      api.getComment({ id }).then(comment => store.replaceComment(comment));
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

    api.vote({ id, url, value: -1 }).then(() => {
      api.getComment({ id }).then(comment => store.replaceComment(comment));
    });
  }

  onReply(...rest) {
    this.props.onReply(...rest);

    this.setState({
      isInputVisible: false,
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
    if (this.props.onCollapseToggle) {
      this.props.onCollapseToggle();
    }
  }

  render(props, { guest, isUserIdVisible, userBlocked, pinned, score, scoreIncreased, scoreDecreased, deleted, isInputVisible }) {
    const { data, mods = {} } = props;
    const isAdmin = !guest && store.get('user').admin;
    const isGuest = guest || !Object.keys(store.get('user')).length;
    const isCurrentUser = (data.user && data.user.id) === (store.get('user') && store.get('user').id);

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
      score: {
        value: Math.abs(score),
        sign: score > 0 ? '+' : (score < 0 ? '−' : null),
        view: score > 0 ? 'positive' : (score < 0 ? 'negative' : null),
      },
      user: {
        ...data.user,
        picture: data.user.picture.indexOf(API_BASE) === 0 ? `${BASE_URL}${data.user.picture}` : data.user.picture,
        isDefaultPicture: !data.user.picture.length,
      },
    };

    const defaultMods = {
      pinned,
      useless: userBlocked || deleted || (score <= USELESS_COMMENT_SCORE && !mods.pinned && !mods.disabled),
      // TODO: add default view mod or don't?
      view: o.user.admin ? 'admin' : null,
      replying: isInputVisible,
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
            <img
              className={b('comment__avatar', {}, { default: o.user.isDefaultPicture })}
              src={o.user.isDefaultPicture ? require('./__avatar/comment__avatar.svg') : o.user.picture}
              alt=""
            />

            <span
              className="comment__username"
              title={o.user.id}
              onClick={this.toggleUserIdVisibility}
            >{o.user.name}</span>

            {
              isUserIdVisible && <span className="comment__user-id"> ({o.user.id})</span>
            }

            <a href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`} className="comment__time">{o.time}</a>

            {
              mods.level > 0 && (
                <a
                  className="comment__link-to-parent"
                  href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.pid}`}
                  aria-label="Go to parent comment"
                  title="Go to parent comment"
                  onClick={this.scrollToParent}
                />
              )
            }

            {
              isAdmin && userBlocked && (
                <span className="comment__status">Blocked</span>
              )
            }

            {
              isAdmin && !userBlocked && deleted && (
                <span className="comment__status">Deleted</span>
              )
            }

            <span className={b('comment__score', {}, { view: o.score.view })}>
              <span
                className={b('comment__vote', {}, { type: 'up', selected: scoreIncreased, disabled: isGuest || isCurrentUser })}
                role="button"
                aria-disabled={isGuest || isCurrentUser}
                tabIndex="0"
                onClick={isGuest || isCurrentUser ? null : this.increaseScore}
                title={isGuest ? 'Only authorized users are allowed to vote' : (isCurrentUser ? 'You can\'t vote for your own comment' : null)}
              >Vote up</span>

              <span className="comment__score-value">
                {o.score.sign}{o.score.value}
              </span>


              <span
                className={b('comment__vote', {}, { type: 'down', selected: scoreDecreased, disabled: isGuest || isCurrentUser })}
                role="button"
                aria-disabled={isGuest || isCurrentUser ? 'true' : 'false'}
                tabIndex="0"
                onClick={isGuest || isCurrentUser ? null : this.decreaseScore}
                title={isGuest ? 'Only authorized users are allowed to vote' : (isCurrentUser ? 'You can\'t vote for your own comment' : null)}
              >Vote down</span>
            </span>
          </div>

          <div
            className={b('comment__text', { mix: 'raw-content' })}
            dangerouslySetInnerHTML={{ __html: o.text }}
          />

          <div className="comment__actions">
            {
              !deleted && !mods.disabled && !isGuest && (
                <span
                  className="comment__action"
                  role="button"
                  tabIndex="0"
                  onClick={this.toggleInputVisibility}
                >{isInputVisible ? 'Cancel' : 'Reply'}</span>
              )
            }

            {
              !mods.disabled && mods.collapsible && (
                <span
                  className={b('comment__action', {}, { type: 'collapse', selected: mods.collapsed })}
                  tabIndex="0"
                  onClick={this.toggleCollapse}
                >{mods.collapsed ? '+' : '−'}</span>
              )
            }

            {
              !deleted && isAdmin && (
                <span className="comment__controls">
                  {
                    !pinned && (
                      <span
                        className="comment__control"
                        role="button"
                        tabIndex="0"
                        onClick={this.onPinClick}
                      >Pin</span>
                    )
                  }

                  {
                    pinned && (
                      <span
                        className="comment__control"
                        role="button"
                        tabIndex="0"
                        onClick={this.onUnpinClick}
                      >Unpin</span>
                    )
                  }

                  {
                    userBlocked && (
                      <span
                        className="comment__control"
                        role="button"
                        tabIndex="0"
                        onClick={this.onUnblockClick}
                      >Unblock</span>
                    )
                  }

                  {
                    !userBlocked && (
                      <span
                        className="comment__control"
                        role="button"
                        tabIndex="0"
                        onClick={this.onBlockClick}
                      >Block</span>
                    )
                  }

                  {
                    !deleted && (
                      <span
                        className="comment__control"
                        role="button"
                        tabIndex="0"
                        onClick={this.onDeleteClick}
                      >Delete</span>
                    )
                  }
                </span>
              )
            }
          </div>
        </div>

        {
          isInputVisible && (
            <Input
              mix="comment__input"
              onSubmit={this.onReply}
              onCancel={this.toggleInputVisibility}
              pid={o.id}
              autoFocus
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
