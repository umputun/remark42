import { h, Component } from 'preact';

import { vote } from 'common/api';
import { url, userId } from 'common/settings';

export default class Comment extends Component {
  constructor(props) {
    super(props);

    const { score, votes } = props.data;

    this.state = {
      score: score,
      scoreIncreased: userId in votes && votes[userId],
      scoreDecreased: userId in votes && !votes[userId],
    };

    this.decreaseScore = this.decreaseScore.bind(this);
    this.increaseScore = this.increaseScore.bind(this);
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

  render(props, { score, scoreIncreased, scoreDecreased }) {
    const { data, mix, mods = {} } = props;
    console.log('rerender', data.id)

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
      <div className={b('comment', props, { view: data.user.admin ? 'admin' : null })} data-id={o.id}>
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
          </div>

          <div className="comment__text" dangerouslySetInnerHTML={{ __html: o.text }}/>
        </div>
      </div>
    );
  }
}
