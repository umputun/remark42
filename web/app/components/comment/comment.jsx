import { h, Component } from 'preact';

import api from 'common/api';
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
    this.onReplyClick = this.onReplyClick.bind(this);
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
    const { pin, score = 0, votes = [] } = props.data;
    const userId = store.get('user').id;

    this.setState({
      score: score,
      pinned: !!pin,
      userBlocked: !!store.get('user').block,
      scoreIncreased: userId in votes && votes[userId],
      scoreDecreased: userId in votes && !votes[userId],
    });
  }

  onReplyClick() {
    const { isInputVisible } = this.state;

    this.setState({ isInputVisible: !isInputVisible });
  }

  onPinClick() {
    const { id } = this.props.data;

    this.setState({ pinned: true });

    api.pin({ id, url }).then(() => {
      api.getComment({ id }).then(comment => store.replaceComment(comment));
    });
  }

  onUnpinClick() {
    const { id } = this.props.data;

    this.setState({ pinned: false });

    api.pin({ id, url }).then(() => {
      api.getComment({ id }).then(comment => store.replaceComment(comment));
    });
  }

  onBlockClick() {
    const { user: { id } } = this.props.data;

    this.setState({ userBlocked: true });

    api.blockUser({ id });
  }

  onUnblockClick() {
    const { user: { id } } = this.props.data;

    this.setState({ userBlocked: false });

    api.unblockUser({ id });
  }

  onDeleteClick() {
    const { id } = this.props.data;

    api.remove({ id }).then(() => {
      if (this.props.onDelete) {
        this.props.onDelete();
      }
    });
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
    this.setState({ isInputVisible: false });
  }

  render(props, { userBlocked, pinned, score, scoreIncreased, scoreDecreased, isInputVisible }) {
    const { data, mix, mods = {} } = props;
    const isAdmin = store.get('user').admin;

    const time = new Date(data.time);
    // TODO: which format for datetime should we choose?
    // TODO: add smth that will count 'hours ago' (mb this: https://github.com/catamphetamine/javascript-time-ago)
    // TODO: check out stash's impl;
    // TODO: don't forget about optional locales, m?
    const timeStr = `${time.toLocaleDateString([], { month: 'short', day: 'numeric' })}, ${time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
    const o = {
      ...data,
      time: timeStr,
      score: {
        value: Math.abs(score),
        sign: score > 0 ? '+' : (score < 0 ? '−' : ''),
      },
    };

    const defaultMods = {
      pinned,
      // TODO: add default view mod or don't?
      view: o.user.admin ? 'admin' : null,
    };

    return (
      <div className={b('comment', props, defaultMods)}>
        <div className="comment__body">
          <img src={o.user.picture} alt="" className="comment__avatar"/>

          <div className="comment__content">
            <div className="comment__info">
              <span className="comment__username">{o.user.name}</span>

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

              <span className="comment__time">{o.time}</span>

              <span className="comment__controls">
                <span className="comment__action" onClick={this.onReplyClick}>reply</span>
              </span>

              {
                isAdmin &&
                (
                  <span className={b('comment__controls', {}, { view: 'admin' })}>

                    {
                      !pinned && (
                        <span className="comment__action" onClick={this.onPinClick}>pin</span>
                      )
                    }

                    {
                      pinned && (
                        <span className="comment__action" onClick={this.onUnpinClick}>unpin</span>
                      )
                    }

                    {
                      userBlocked && (
                        <span className="comment__action" onClick={this.onUnblockClick}>unblock</span>
                      )
                    }

                    {
                      !userBlocked && (
                        <span className="comment__action" onClick={this.onBlockClick}>block</span>
                      )
                    }

                    <span className="comment__action" onClick={this.onDeleteClick}>delete</span>
                  </span>
                )
              }
            </div>

            <div className="comment__text" dangerouslySetInnerHTML={{ __html: o.text }}/>
          </div>
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
