import { h, Component } from 'preact';

import { DEFAULT_MAX_COMMENT_SIZE } from 'common/constants';

import api from 'common/api';
import store from 'common/store';

export default class Input extends Component {
  constructor(props) {
    super(props);

    const config = store.get('config') || {};

    this.state = {
      preview: null,
      isErrorShown: false,
      maxLength: config.max_comment_size || DEFAULT_MAX_COMMENT_SIZE,
      commentLength: 0,
    };

    this.send = this.send.bind(this);
    this.getPreview = this.getPreview.bind(this);
    this.onInput = this.onInput.bind(this);
    this.onKeyDown = this.onKeyDown.bind(this);
  }

  componentDidMount() {
    if (this.props.autoFocus) {
      this.fieldNode.focus();
    }

    this.fieldNode.value = '';

    store.onUpdate('config', config => {
      this.setState({ maxLength: config && config.max_comment_size || DEFAULT_MAX_COMMENT_SIZE });
    });
  }

  onKeyDown(e) {
    // send on cmd+enter / ctrl+enter
    if (e.keyCode === 13 && (e.metaKey || e.ctrlKey)) {
      this.send();
    }

    // cancel on esc
    if (e.keyCode === 27 && this.props.onCancel) {
      this.props.onCancel();
    }
  }

  onInput() {
    this.autoResize();

    this.setState({
      preview: null,
      isErrorShown: false,
      commentLength: this.fieldNode.value.length,
    });
  }

  autoResize() {
    this.fieldNode.style.height = '';
    this.fieldNode.style.height = `${this.fieldNode.scrollHeight}px`;
  }

  send(e) {
    const text = this.fieldNode.value;
    const pid = this.props.pid;

    if (e) e.preventDefault();

    if (!text || !text.trim()) return;

    this.setState({ isFieldDisabled: true, isErrorShown: false });

    api.send({ text, ...(pid ? { pid } : {}) })
      .then(({ id }) => {
        // TODO: maybe we should run onsubmit before send; like in optimistic ui
        if (this.props.onSubmit) {
          this.props.onSubmit({ text, id, pid });
        }

        this.fieldNode.value = '';
        this.fieldNode.style.height = '';
        this.setState({ preview: null });
      })
      .catch(() => {
        this.setState({ isErrorShown: true });
      })
      .finally(() => this.setState({ isFieldDisabled: false }));
  }

  getPreview() {
    const text = this.fieldNode.value;

    if (!text || !text.trim()) return;

    this.setState({ isErrorShown: false });

    api.getPreview({ text })
      .then(preview => this.setState({ preview }))
      .catch(() => {
        this.setState({ isErrorShown: true });
      });
  }

  render(props, { isFieldDisabled, isErrorShown, preview, maxLength, commentLength }) {
    const charactersLeft = maxLength - commentLength;

    return (
      <form className={b('input', props)} onSubmit={this.send} role="form" aria-label="New Comment">
        <div className="input__field-wrapper">
          <textarea
            className="input__field"
            placeholder="Your comment here"
            maxLength={maxLength}
            onInput={this.onInput}
            onKeyDown={this.onKeyDown}
            ref={r => (this.fieldNode = r)}
            required
            disabled={isFieldDisabled}
          />

          {
            (charactersLeft < 100) && (
              <span className="input__counter">{charactersLeft}</span>
            )
          }
        </div>

        {
          isErrorShown && <p className="input__error" role="alert">Something went wrong. Please try again a bit later.</p>
        }

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
