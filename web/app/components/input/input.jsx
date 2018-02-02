import { h, Component } from 'preact';

import api from 'common/api';

export default class Input extends Component {
  constructor(props) {
    super(props);

    this.autoResize = this.autoResize.bind(this);
    this.send = this.send.bind(this);
  }

  componentDidMount() {
    if (this.props.autoFocus) {
      this.fieldNode.focus();
    }
  }

  autoResize() {
    this.fieldNode.style.height = '';
    this.setState({ height: this.fieldNode.scrollHeight });
  }

  send(e) {
    const text = this.fieldNode.value;
    const pid = this.props.pid;

    e.preventDefault();
    api.send({ text, ...(pid ? { pid } : {}) })
      .then(({ id }) => {
        // TODO: maybe we should run onsubmit before send; like in optimistic ui
        if (this.props.onSubmit) {
          this.props.onSubmit({ text, id, pid });
        }

        this.fieldNode.value = '';
      })
      .catch(() => {
        // TODO: do smth?
      });
  }

  render(props, { height }) {
    return (
      <form className={b('input', props)} onSubmit={this.send}>
        <textarea
          className="input__field"
          onInput={this.autoResize}
          style={{ height }}
          ref={r => (this.fieldNode = r)}
          required
        >
          {props.children}
        </textarea>

        <button className="input__button" type="submit">Send</button>
      </form>
    );
  }
}
