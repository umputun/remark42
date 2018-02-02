import { h, Component } from 'preact';

import { vote } from 'common/api';
import { url } from 'common/settings';
import store from 'common/store';

import Input from 'components/input';

export default class Comment extends Component {
  constructor(props) {
    super(props);

    const { score = 0, votes = [] } = props.data;
    const userId = store.get('user').id;

    this.state = {
      score: score,
      scoreIncreased: userId in votes && votes[userId],
      scoreDecreased: userId in votes && !votes[userId],
      isInputVisible: false,
    };

    this.decreaseScore = this.decreaseScore.bind(this);
    this.increaseScore = this.increaseScore.bind(this);
    this.onReplyClick = this.onReplyClick.bind(this);
    this.onReply = this.onReply.bind(this);
  }

  onReplyClick() {
    const { isInputVisible } = this.state;

    this.setState({ isInputVisible: !isInputVisible });
  }

  increaseScore() {
    const { score, scoreIncreased, scoreDecreased } = this.state;

    if (scoreIncreased) return;

    this.setState({
      scoreIncreased: !scoreDecreased,
      scoreDecreased: false,
      score: score + 1,
    });

    vote({ id: this.props.data.id, url, value: 1 });
  }

  decreaseScore() {
    const { score, scoreIncreased, scoreDecreased } = this.state;

    if (scoreDecreased) return;

    this.setState({
      scoreDecreased: !scoreIncreased,
      scoreIncreased: false,
      score: score - 1,
    });

    vote({ id: this.props.data.id, url, value: -1 });
  }

  onReply(...rest) {
    this.props.onReply(...rest);
    this.setState({ isInputVisible: false });
  }

  render(props, { score, scoreIncreased, scoreDecreased, isInputVisible }) {
    const { data, mix, mods = {} } = props;

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

    return (
      // TODO: add default view mod or don't?
      <div className={b('comment', props, { view: data.user.admin ? 'admin' : null })}>
        <div className="comment__body">
          <img src={o.user.picture} alt="" className="comment__avatar"/>

          <div className="comment__content">
            <div className="comment__info">
              <a href="#" className="comment__username">{o.user.name}</a>

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
              <span className="comment__reply" onClick={this.onReplyClick}>reply</span>
            </span>
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
