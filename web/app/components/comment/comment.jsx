import { h, Component } from 'preact';

import api from 'common/api';
import { API_BASE, BASE_URL } from 'common/constants';
import { url } from 'common/settings';
import store from 'common/store';

import Input from 'components/input';

export default class Comment extends Component {
  constructor(props) {
    super(props);

    this.state = {
      isInputVisible: false,
    };

    this.updateState(props);

    this.decreaseScore = this.decreaseScore.bind(this);
    this.increaseScore = this.increaseScore.bind(this);
    this.toggleInputVisibility = this.toggleInputVisibility.bind(this);
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

    this.setState({ isInputVisible: !isInputVisible });

    if (this.props.onReplyClick) {
      if (!isInputVisible) {
        this.props.onReplyClick(() => this.setState({ isInputVisible: false }));
      } else {
        this.props.onReplyClick(null);
      }
    }
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
  }

  render(props, { guest, userBlocked, pinned, score, scoreIncreased, scoreDecreased, deleted, isInputVisible }) {
    const { data, mix, mods = {} } = props;
    const isAdmin = !guest && store.get('user').admin;
    const isGuest = guest || !Object.keys(store.get('user')).length;

    const time = new Date(data.time);
    // TODO: which format for datetime should we choose?
    // TODO: add smth that will count 'hours ago' (mb this: https://github.com/catamphetamine/javascript-time-ago)
    // TODO: check out stash's impl;
    // TODO: don't forget about optional locales, m?
    // TODO: also date must be a link to the comment
    const timeStr = `${time.toLocaleDateString([], { month: 'short', day: 'numeric' })}, ${time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
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
      time: timeStr,
      score: {
        value: Math.abs(score),
        sign: score > 0 ? '+' : (score < 0 ? '−' : ''),
      },
      user: {
        ...data.user,
        picture: data.user.picture.indexOf(API_BASE) === 0 ? `${BASE_URL}${data.user.picture}` : data.user.picture,
      },
    };

    const defaultMods = {
      pinned,
      useless: userBlocked || deleted,
      // TODO: add default view mod or don't?
      view: o.user.admin ? 'admin' : null,
      replying: isInputVisible,
    };

    if (mods.view === 'preview') {
      return (
        <div className={b('comment', props, defaultMods)} id={`remark__comment-${o.id}`}>
          <div className="comment__body">
            <div className="comment__info">
              <a href={`${o.locator.url}#remark__comment-${o.id}`} className="comment__username">{o.user.name}</a>
            </div>
            {' '}
            <div className="comment__text" dangerouslySetInnerHTML={{ __html: o.text }}/>
          </div>
        </div>
      );
    }

    return (
      <div className={b('comment', props, defaultMods)} id={`remark__comment-${o.id}`}>
        <div className="comment__body">
          <div className="comment__info">
            <img src={o.user.picture} alt="" className="comment__avatar"/>

            <span className="comment__username">{o.user.name}</span>

            {
              !isGuest && (
                <span className="comment__score">
                  <span
                    className={b('comment__vote', {}, { type: 'up', selected: scoreIncreased })}
                    onClick={this.increaseScore}
                  >vote up</span>

                  <span className="comment__score-sign">{o.score.sign}</span>

                  <span className="comment__score-value">{o.score.value}</span>

                  <span
                    className={b('comment__vote', {}, { type: 'down', selected: scoreDecreased })}
                    onClick={this.decreaseScore}
                  >vote down</span>
                </span>
              )
            }

            <span className="comment__time">{o.time}</span>

            {
              isAdmin && userBlocked && (
                <span className="comment__status">blocked</span>
              )
            }

            {
              isAdmin && !userBlocked && deleted && (
                <span className="comment__status">deleted</span>
              )
            }

            {
              !mods.disabled && !isGuest && (
                <span className="comment__controls">
                  <span
                    className="comment__action"
                    tabIndex="0"
                    onClick={this.toggleInputVisibility}
                  >reply</span>
                </span>
              )
            }

            {
              isAdmin &&
              (
                <span className={b('comment__controls', {}, { view: 'admin' })}>

                  {
                    !pinned && (
                      <span
                        className="comment__action"
                        tabIndex="0"
                        onClick={this.onPinClick}
                      >pin</span>
                    )
                  }

                  {
                    pinned && (
                      <span
                        className="comment__action"
                        tabIndex="0"
                        onClick={this.onUnpinClick}
                      >unpin</span>
                    )
                  }

                  {
                    userBlocked && (
                      <span
                        className="comment__action"
                        tabIndex="0"
                        onClick={this.onUnblockClick}
                      >unblock</span>
                    )
                  }

                  {
                    !userBlocked && (
                      <span
                        className="comment__action"
                        tabIndex="0"
                        onClick={this.onBlockClick}
                      >block</span>
                    )
                  }

                  {
                    !deleted && (
                      <span
                        className="comment__action"
                        tabIndex="0"
                        onClick={this.onDeleteClick}
                      >delete</span>
                    )
                  }
                </span>
              )
            }
          </div>

          <div className="comment__text" dangerouslySetInnerHTML={{ __html: o.text }}/>
        </div>

        {
          isInputVisible && (
            <Input mix="comment__input" onSubmit={this.onReply} pid={o.id} autoFocus/>
          )
        }
      </div>
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
