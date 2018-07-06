/** @jsx h */
import { h, Component } from 'preact';

import { BASE_URL, API_BASE, DEFAULT_MAX_COMMENT_SIZE } from 'common/constants';
import { siteId, url } from 'common/settings';

import api from 'common/api';
import store from 'common/store';

const RSS_THREAD_URL = `${BASE_URL}${API_BASE}/rss/post?site=${siteId}&url=${url}`;
const RSS_SITE_URL = `${BASE_URL}${API_BASE}/rss/site?site=${siteId}`;

export default class Input extends Component {
  constructor(props) {
    super(props);

    const config = store.get('config') || {};

    this.state = {
      preview: null,
      isErrorShown: false,
      isDisabled: false,
      maxLength: config.max_comment_size || DEFAULT_MAX_COMMENT_SIZE,
      commentLength: 0,
    };

    this.send = this.send.bind(this);
    this.getPreview = this.getPreview.bind(this);
    this.onInput = this.onInput.bind(this);
    this.onKeyDown = this.onKeyDown.bind(this);
  }

  componentDidMount() {
    const { mods = {}, value } = this.props;

    if (this.props.autoFocus) {
      this.fieldNode.focus();
    }

    if (mods.mode !== 'edit') {
      this.fieldNode.value = '';
    } else {
      this.fieldNode.value = value;
      this.autoResize();
    }

    store.onUpdate('config', config => {
      this.setState({ maxLength: (config && config.max_comment_size) || DEFAULT_MAX_COMMENT_SIZE });
    });
  }

  componentWillUnmount() {
    this.fieldNode.value = '';
  }

  shouldComponentUpdate(nextProps, nextState) {
    return (
      nextProps.id !== this.props.id ||
      nextProps.pid !== this.props.pid ||
      nextProps.value !== this.props.value ||
      nextProps.errorMessage !== this.props.errorMessage ||
      nextState !== this.state
    );
  }

  onKeyDown(e) {
    // send on cmd+enter / ctrl+enter
    if (e.keyCode === 13 && (e.metaKey || e.ctrlKey)) {
      this.send();
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
    const { mods = {}, pid, id } = this.props;

    if (e) e.preventDefault();

    if (!text || !text.trim()) return;

    this.setState({ isDisabled: true, isErrorShown: false });

    const request =
      mods.mode === 'edit' ? api.updateComment({ text, id }) : api.addComment({ text, ...(pid ? { pid } : {}) });

    request
      .then(comment => {
        if (this.props.onSubmit) {
          this.props.onSubmit(comment);
        }

        this.fieldNode.value = '';
        this.fieldNode.style.height = '';
        this.setState({ preview: null });
      })
      .catch(() => {
        this.setState({ isErrorShown: true });
      })
      .finally(() => this.setState({ isDisabled: false }));
  }

  getPreview() {
    const text = this.fieldNode.value;

    if (!text || !text.trim()) return;

    this.setState({ isErrorShown: false });

    api
      .getPreview({ text })
      .then(preview => this.setState({ preview }))
      .catch(() => {
        this.setState({ isErrorShown: true });
      });
  }

  render(props, { isDisabled, isErrorShown, preview, maxLength, commentLength }) {
    const charactersLeft = maxLength - commentLength;
    const { mods = {}, value = null, errorMessage } = props;

    return (
      <form className={b('input', props)} onSubmit={this.send} role="form" aria-label="New comment">
        <div className="input__field-wrapper">
          <textarea
            className="input__field"
            placeholder="Your comment here"
            defaultValue={value}
            maxLength={maxLength}
            onInput={this.onInput}
            onKeyDown={this.onKeyDown}
            ref={r => (this.fieldNode = r)}
            disabled={isDisabled}
          />

          {charactersLeft < 100 && <span className="input__counter">{charactersLeft}</span>}
        </div>

        {(isErrorShown || !!errorMessage) && (
          <p className="input__error" role="alert">
            {errorMessage || 'Something went wrong. Please try again a bit later.'}
          </p>
        )}

        <div className="input__actions">
          <button
            className={b('input__button', {}, { type: 'preview' })}
            type="button"
            disabled={isDisabled}
            onClick={this.getPreview}
          >
            Preview
          </button>

          <button className={b('input__button', {}, { type: 'send' })} type="submit" disabled={isDisabled}>
            Send
          </button>

          {mods.type === 'main' && (
            <div className="input__rss">
              Subscribe to&nbsp;this{' '}
              <a className="input__rss-link" href={RSS_THREAD_URL} target="_blank">
                Thread
              </a>{' '}
              or&nbsp;
              <a className="input__rss-link" href={RSS_SITE_URL} target="_blank">
                Site
              </a>{' '}
              by&nbsp;RSS
            </div>
          )}
        </div>

        {// TODO: it can be more elegant;
        // for example it can render full comment component here (or above textarea on mobile)
        !!preview && (
          <div className="input__preview-wrapper">
            <div
              className={b('input__preview', { mix: 'raw-content' })}
              dangerouslySetInnerHTML={{ __html: preview }}
            />
          </div>
        )}
      </form>
    );
  }
}
