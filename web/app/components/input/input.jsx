/** @jsx h */
import { h, Component } from 'preact';

import { BASE_URL, API_BASE, DEFAULT_MAX_COMMENT_SIZE } from 'common/constants';
import { siteId, url } from 'common/settings';
import { saveTimeDiff } from 'common/comments';

import api from 'common/api';
import store from 'common/store';
import TextareaAutosize from 'components/input/textarea-autosize';

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
      text: props.value || '',
    };

    this.send = this.send.bind(this);
    this.getPreview = this.getPreview.bind(this);
    this.onInput = this.onInput.bind(this);
    this.onKeyDown = this.onKeyDown.bind(this);
  }

  componentDidMount() {
    store.onUpdate('config', config => {
      this.setState({ maxLength: (config && config.max_comment_size) || DEFAULT_MAX_COMMENT_SIZE });
    });
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

  onInput(e) {
    this.setState({
      preview: null,
      isErrorShown: false,
      text: e.target.value,
    });
  }

  send(e) {
    const text = this.state.text;
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

        // save time defferese between client & server
        const timeDiff =
          comment.dateHeader && comment.dateHeader.length > 0 ? (new Date() - new Date(comment.dateHeader)) / 1000 : 0;
        comment.timeDiff = timeDiff;
        if (comment.replies) {
          saveTimeDiff(comment.replies, timeDiff);
        }

        this.setState({ preview: null, text: '' });
      })
      .catch(() => {
        this.setState({ isErrorShown: true });
      })
      .finally(() => this.setState({ isDisabled: false }));
  }

  getPreview() {
    const text = this.state.text;

    if (!text || !text.trim()) return;

    this.setState({ isErrorShown: false });

    api
      .getPreview({ text })
      .then(preview => this.setState({ preview }))
      .catch(() => {
        this.setState({ isErrorShown: true });
      });
  }

  render(props, { isDisabled, isErrorShown, preview, maxLength, text }) {
    const charactersLeft = maxLength - text.length;
    const { mods = {}, errorMessage } = props;

    return (
      <form className={b('input', props)} onSubmit={this.send} aria-label="New comment">
        <div className="input__field-wrapper">
          <TextareaAutosize
            className="input__field"
            placeholder="Your comment here"
            value={text}
            maxLength={maxLength}
            onInput={this.onInput}
            onKeyDown={this.onKeyDown}
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
