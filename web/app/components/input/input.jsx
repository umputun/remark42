import { h, Component } from 'preact';

import api from 'common/api';

export default class Input extends Component {
  constructor(props) {
    super(props);

    this.autoResize = this.autoResize.bind(this);
    this.send = this.send.bind(this);
    this.onKeyDown = this.onKeyDown.bind(this);
  }

  componentDidMount() {
    if (this.props.autoFocus) {
      this.fieldNode.focus();
    }
  }

  onKeyDown(e) {
    if (e.keyCode === 13 && (e.metaKey || e.ctrlKey)) {
      this.send();
    }
  }

  autoResize() {
    this.fieldNode.style.height = '';
    this.setState({ height: this.fieldNode.scrollHeight });
  }

  send(e) {
    const text = this.fieldNode.value;
    const pid = this.props.pid;

    if (e) e.preventDefault();

    if (!text || !text.trim()) return;

    this.setState({ isFieldDisabled: true });

    api.send({ text, ...(pid ? { pid } : {}) })
      .then(({ id }) => {
        // TODO: maybe we should run onsubmit before send; like in optimistic ui
        if (this.props.onSubmit) {
          this.props.onSubmit({ text, id, pid });
        }

        this.fieldNode.value = '';
        this.setState({ height: null });
      })
      .catch(() => {
        // TODO: do smth?
      })
      .finally(() => this.setState({ isFieldDisabled: false }));
  }

  render(props, { height, isFieldDisabled }) {
    return (
      <form className={b('input', props)} onSubmit={this.send}>
        <textarea
          className="input__field"
          onInput={this.autoResize}
          onKeyDown={this.onKeyDown}
          style={{ height }}
          ref={r => (this.fieldNode = r)}
          required
          disabled={isFieldDisabled}
        >
          {props.children}
        </textarea>

        <button className="input__button" type="submit">Send</button>
      </form>
    );
  }
}
