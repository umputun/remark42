import { h, Component } from 'preact';

import api from 'common/api';

export default class Input extends Component {
  constructor(props) {
    super(props);

    this.state = {
      preview: null,
    };

    this.autoResize = this.autoResize.bind(this);
    this.send = this.send.bind(this);
    this.getPreview = this.getPreview.bind(this);
    this.onKeyDown = this.onKeyDown.bind(this);
  }

  componentDidMount() {
    if (this.props.autoFocus) {
      this.fieldNode.focus();
    }

    this.fieldNode.value = '';
  }

  onKeyDown(e) {
    if (e.keyCode === 13 && (e.metaKey || e.ctrlKey)) {
      this.send();
    }
  }

  autoResize() {
    this.fieldNode.style.height = '';
    this.setState({
      height: this.fieldNode.scrollHeight,
      preview: null,
    });
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
        this.setState({ height: null, preview: null });
      })
      .catch(() => {
        // TODO: do smth?
      })
      .finally(() => this.setState({ isFieldDisabled: false }));
  }

  getPreview() {
    const text = this.fieldNode.value;

    if (!text || !text.trim()) return;

    api.getPreview({ text })
      .then(preview => this.setState({ preview }))
      .catch(() => {
        // TODO: do smth?
      });
  }

  render(props, { height, isFieldDisabled, preview }) {
    return (
      <form className={b('input', props)} onSubmit={this.send}>
        <textarea
          className="input__field"
          placeholder="Your comment here"
          onInput={this.autoResize}
          onKeyDown={this.onKeyDown}
          style={{ height }}
          ref={r => (this.fieldNode = r)}
          required
          disabled={isFieldDisabled}
        >
          {props.children}
        </textarea>

        <div className="input__buttons">
          <button
            className={b('input__button', {}, { type: 'preview' })}
            type="button"
            onClick={this.getPreview}
          >Preview</button>

          <button
            className={b('input__button', {}, { type: 'send' })}
            type="submit"
          >Send</button>
        </div>

        {
          // TODO: it can be more elegant;
          // for example it can render full comment component here (or above textarea on mobile)
          !!preview && (
            <div
              className={b('input__preview', { mix: 'raw-content' })}
              dangerouslySetInnerHTML={{ __html: preview }}
            />
          )
        }
      </form>
    );
  }
}
