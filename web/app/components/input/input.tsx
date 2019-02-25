/** @jsx h */

/* styles imports */
import '@app/components/raw-content';
import './styles';

import { h, Component, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { User, Comment, Theme } from '@app/common/types';
import { BASE_URL, API_BASE } from '@app/common/constants';
import { StaticStore } from '@app/common/static_store';
import { siteId, url, pageTitle } from '@app/common/settings';
import { extractErrorMessageFromResponse } from '@app/utils/errorUtils';

import TextareaAutosize from './textarea-autosize';

const RSS_THREAD_URL = `${BASE_URL}${API_BASE}/rss/post?site=${siteId}&url=${url}`;
const RSS_SITE_URL = `${BASE_URL}${API_BASE}/rss/site?site=${siteId}`;
const RSS_REPLIES_URL = `${BASE_URL}${API_BASE}/rss/reply?site=${siteId}&user=`;

interface Props {
  id?: Comment['id'];
  pid?: Comment['id'];
  userId?: User['id'];
  errorMessage?: string;
  value?: string;
  mix?: string;
  mode?: 'main' | 'edit' | 'reply';
  theme: Theme;
  autofocus?: boolean;

  onSubmit(text: string, pageTitle: string): Promise<void>;
  getPreview(text: string): Promise<string>;
  /** action on cancel. optional as root input has no cancel option */
  onCancel?: () => void;
}

interface State {
  preview: string | null;
  isErrorShown: boolean;
  errorMessage: string | null;
  isDisabled: boolean;
  maxLength: number;
  text: string;
}

export class Input extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      preview: null,
      isErrorShown: false,
      errorMessage: null,
      isDisabled: false,
      maxLength: StaticStore.config.max_comment_size,
      text: props.value || '',
    };

    this.send = this.send.bind(this);
    this.getPreview = this.getPreview.bind(this);
    this.onInput = this.onInput.bind(this);
    this.onKeyDown = this.onKeyDown.bind(this);
  }

  shouldComponentUpdate(nextProps: Props, nextState: State) {
    return (
      nextProps.id !== this.props.id ||
      nextProps.mode !== this.props.mode ||
      nextProps.theme !== this.props.theme ||
      nextProps.userId !== this.props.userId ||
      nextProps.pid !== this.props.pid ||
      nextProps.value !== this.props.value ||
      nextProps.errorMessage !== this.props.errorMessage ||
      nextState !== this.state
    );
  }

  onKeyDown(e: KeyboardEvent) {
    // send on cmd+enter / ctrl+enter
    if (e.keyCode === 13 && (e.metaKey || e.ctrlKey)) {
      this.send(e);
    }
  }

  onInput(e: Event) {
    this.setState({
      preview: null,
      isErrorShown: false,
      errorMessage: null,
      text: (e.target as HTMLInputElement).value,
    });
  }

  send(e: Event) {
    const text = this.state.text;
    const props = this.props;

    if (e) e.preventDefault();

    if (!text || !text.trim()) return;

    if (text === this.props.value) {
      this.props.onCancel && this.props.onCancel();
      this.setState({ preview: null, text: '' });
    }

    this.setState({ isDisabled: true, isErrorShown: false });

    props
      .onSubmit(text, pageTitle || document.title)
      .then(() => {
        this.setState({ preview: null, text: '' });
      })
      .catch(e => {
        console.error(e); // eslint-disable-line no-console
        const errorMessage = extractErrorMessageFromResponse(e);
        this.setState({ isErrorShown: true, errorMessage });
      })
      .finally(() => this.setState({ isDisabled: false }));
  }

  getPreview() {
    const text = this.state.text;

    if (!text || !text.trim()) return;

    this.setState({ isErrorShown: false, errorMessage: null });

    this.props
      .getPreview(text)
      .then(preview => this.setState({ preview }))
      .catch(() => {
        this.setState({ isErrorShown: true, errorMessage: null });
      });
  }

  render(props: RenderableProps<Props>, { isDisabled, isErrorShown, errorMessage, preview, maxLength, text }: State) {
    const charactersLeft = maxLength - text.length;
    errorMessage = props.errorMessage || errorMessage;

    return (
      <form
        className={b('input', {
          mods: {
            theme: props.theme || 'light',
            type: props.mode || 'reply',
          },
          mix: props.mix,
        })}
        onSubmit={this.send}
        aria-label="New comment"
      >
        <div className="input__field-wrapper">
          <TextareaAutosize
            className="input__field"
            placeholder="Your comment here"
            value={text}
            maxLength={maxLength}
            onInput={this.onInput}
            onKeyDown={this.onKeyDown}
            disabled={isDisabled}
            autofocus={!!props.autofocus}
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

          {props.mode === 'main' && (
            <div className="input__rss">
              <div class="input__markdown">
                Styling with{' '}
                <a className="input__markdown-link" target="_blank" href="markdown-help.html">
                  Markdown
                </a>{' '}
                is supported
              </div>
              Subscribe to&nbsp;the{' '}
              <a className="input__rss-link" href={RSS_THREAD_URL} target="_blank">
                Thread
              </a>
              {', '}
              <a className="input__rss-link" href={RSS_SITE_URL} target="_blank">
                Site
              </a>{' '}
              or&nbsp;
              <a className="input__rss-link" href={RSS_REPLIES_URL + props.userId} target="_blank">
                Replies
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
              className={b('input__preview', { mix: b('raw-content', {}, { theme: props.theme }) })}
              dangerouslySetInnerHTML={{ __html: preview }}
            />
          </div>
        )}
      </form>
    );
  }
}
