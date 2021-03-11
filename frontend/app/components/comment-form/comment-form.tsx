import { h, Component, createRef, Fragment } from 'preact';
import { FormattedMessage, IntlShape, defineMessages } from 'react-intl';
import b, { Mix } from 'bem-react-helper';

import { User, Theme, Image, ApiError } from 'common/types';
import { StaticStore } from 'common/static-store';
import { pageTitle } from 'common/settings';
import { extractErrorMessageFromResponse } from 'utils/errorUtils';
import { isUserAnonymous } from 'utils/isUserAnonymous';
import { sleep } from 'utils/sleep';
import { replaceSelection } from 'utils/replaceSelection';
import { Button } from 'components/button';
import TextareaAutosize from 'components/textarea-autosize';
import Auth from 'components/auth';
import { getJsonItem, updateJsonItem } from 'common/local-storage';
import { LS_SAVED_COMMENT_VALUE } from 'common/constants';

import { SubscribeByEmail } from './__subscribe-by-email';
import { SubscribeByRSS } from './__subscribe-by-rss';

import MarkdownToolbar from './markdown-toolbar';
import { TextExpander } from './text-expander';

let textareaId = 0;

export type CommentFormProps = {
  id: string;
  user: User | null;
  errorMessage?: string;
  value?: string;
  mix?: Mix;
  mode?: 'main' | 'edit' | 'reply';
  theme: Theme;
  simpleView?: boolean;
  autofocus?: boolean;

  onSubmit(text: string, pageTitle: string): Promise<void>;
  getPreview(text: string): Promise<string>;
  /** action on cancel. optional as root input has no cancel option */
  onCancel?: () => void;
  uploadImage?: (image: File) => Promise<Image>;
  intl: IntlShape;
};

export type CommentFormState = {
  preview: string | null;
  isErrorShown: boolean;
  /** error message, if contains newlines, it will be splitted to multiple errors */
  errorMessage: string | null;
  /** prevents error hiding on input event */
  errorLock: boolean;
  isDisabled: boolean;
  /** main input value */
  text: string;
  /** override main button text */
  buttonText: null | string;
};

const ImageMimeRegex = /image\//i;

export const messages = defineMessages({
  placeholder: {
    id: 'commentForm.input-placeholder',
    defaultMessage: 'Your comment here',
  },
  uploadFileFail: {
    id: 'commentForm.upload-file-fail',
    defaultMessage: '{fileName} upload failed with "{errorMessage}"',
  },
  uploading: {
    id: 'commentForm.uploading',
    defaultMessage: 'Uploading...',
  },
  uploadingFile: {
    id: 'commentForm.uploading-file',
    defaultMessage: 'uploading {fileName}...',
  },
  exceededSize: {
    id: 'commentForm.exceeded-size',
    defaultMessage: '{fileName} exceeds size limit of {maxImageSize}',
  },
  newComment: {
    id: 'commentForm.new-comment',
    defaultMessage: 'New comment',
  },
  unexpectedError: {
    id: 'commentForm.unexpected-error',
    defaultMessage: 'Something went wrong. Please try again a bit later.',
  },
  unauthorizedUploadingDisabled: {
    id: 'commentForm.unauthorized-uploading-disabled',
    defaultMessage: 'Image uploading is disabled for unauthorized users. You should login before uploading.',
  },
  anonymousUploadingDisabled: {
    id: 'commentForm.anonymous-uploading-disabled',
    defaultMessage:
      'Image uploading is disabled for anonymous users. Please log in not as anonymous user to be able to attach images.',
  },
});

export class CommentForm extends Component<CommentFormProps, CommentFormState> {
  /** reference to textarea element */
  textareaRef = createRef<HTMLTextAreaElement>();
  textareaId: string;

  constructor(props: CommentFormProps) {
    super(props);
    textareaId = textareaId + 1;
    this.textareaId = `textarea_${textareaId}`;

    const savedComments = getJsonItem<Record<string, string>>(LS_SAVED_COMMENT_VALUE);
    let text = savedComments?.[props.id] ?? '';

    if (props.value) {
      text = props.value;
    }

    this.state = {
      preview: null,
      isErrorShown: false,
      errorMessage: null,
      errorLock: false,
      isDisabled: false,
      text,
      buttonText: null,
    };

    this.getPreview = this.getPreview.bind(this);
    this.onKeyDown = this.onKeyDown.bind(this);
    this.onDragOver = this.onDragOver.bind(this);
    this.onDrop = this.onDrop.bind(this);
    this.appendError = this.appendError.bind(this);
    this.uploadImage = this.uploadImage.bind(this);
    this.uploadImages = this.uploadImages.bind(this);
    this.onPaste = this.onPaste.bind(this);
  }

  componentWillReceiveProps(nextProps: CommentFormProps) {
    if (nextProps.value !== this.props.value) {
      this.setState({ text: nextProps.value || '' });
    }
    if (nextProps.user && !this.props.value) {
      this.setState({
        isErrorShown: false,
        errorMessage: null,
      });
    }
  }

  shouldComponentUpdate(nextProps: CommentFormProps, nextState: CommentFormState) {
    const userId = this.props.user !== null && this.props.user.id;
    const nextUserId = nextProps.user !== null && nextProps.user.id;

    return (
      nextUserId !== userId ||
      nextProps.mode !== this.props.mode ||
      nextProps.theme !== this.props.theme ||
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

  onInput = (e: Event) => {
    const { value } = e.target as HTMLInputElement;
    const text = value.substr(0, StaticStore.config.max_comment_size);

    updateJsonItem(LS_SAVED_COMMENT_VALUE, { [this.props.id]: value });

    if (this.state.errorLock) {
      this.setState({
        preview: null,
        text,
      });
      return;
    }

    this.setState({
      isErrorShown: false,
      errorMessage: null,
      preview: null,
      text,
    });
  };

  async onPaste(e: ClipboardEvent) {
    if (!(e.clipboardData && e.clipboardData.files.length > 0)) {
      return;
    }
    e.preventDefault();
    const files = Array.from(e.clipboardData.files);
    await this.uploadImages(files);
  }

  send = async (e: Event) => {
    const { text } = this.state;

    if (e) e.preventDefault();

    if (!text || !text.trim()) return;
    if (text === this.props.value) {
      this.props.onCancel && this.props.onCancel();
      this.setState({ preview: null, text: '' });
    }

    this.setState({ isDisabled: true, isErrorShown: false, text });
    try {
      await this.props.onSubmit(text, pageTitle || document.title);
    } catch (e) {
      this.setState({
        isErrorShown: true,
        errorMessage: extractErrorMessageFromResponse(e, this.props.intl),
      });
      return;
    }

    updateJsonItem<Record<string, string> | null>(LS_SAVED_COMMENT_VALUE, (data) => {
      if (data === null) {
        return null;
      }
      delete data[this.props.id];
      return data;
    });
    this.setState({ isDisabled: false, preview: null, text: '' });
  };

  getPreview() {
    const text = this.textareaRef.current?.value ?? this.state.text;

    if (!text || !text.trim()) return;

    this.setState({ isErrorShown: false, errorMessage: null, text });

    this.props
      .getPreview(text)
      .then((preview) => this.setState({ preview }))
      .catch(() => {
        this.setState({ isErrorShown: true, errorMessage: null });
      });
  }

  /** appends error to input's error block */
  appendError(...errors: string[]) {
    if (!this.state.errorMessage) {
      this.setState({
        errorMessage: errors.join('\n'),
        isErrorShown: true,
      });
      return;
    }
    this.setState({
      errorMessage: `${this.state.errorMessage}\n${errors.join('\n')}`,
      isErrorShown: true,
    });
  }

  onDragOver(e: DragEvent) {
    if (!this.props.user) e.preventDefault();
    if (!this.props.uploadImage) return;
    if (StaticStore.config.max_image_size === 0) return;
    if (!this.textareaRef.current) return;
    if (!e.dataTransfer) return;
    const items = Array.from(e.dataTransfer.items);
    if (Array.from(items).filter((i) => i.kind === 'file' && ImageMimeRegex.test(i.type)).length === 0) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
  }

  onDrop(e: DragEvent) {
    const isAnonymous = this.props.user && isUserAnonymous(this.props.user);
    if (!this.props.user || isAnonymous) {
      const message = isAnonymous ? messages.anonymousUploadingDisabled : messages.unauthorizedUploadingDisabled;
      this.setState({
        isErrorShown: true,
        errorMessage: this.props.intl.formatMessage(message),
      });
      e.preventDefault();
      return;
    }
    if (!this.props.uploadImage) return;
    if (StaticStore.config.max_image_size === 0) return;
    if (!e.dataTransfer) return;

    const data = Array.from(e.dataTransfer.files).filter((f) => ImageMimeRegex.test(f.type));
    if (data.length === 0) return;

    e.preventDefault();

    this.uploadImages(data);
  }

  /** returns selection range of a textarea */
  getSelection(): [number, number] {
    const textarea = this.textareaRef.current;

    if (textarea) {
      return [textarea.selectionStart, textarea.selectionEnd];
    }

    throw new Error('No textarea element reference exists');
  }

  /** sets selection range of a textarea */
  setSelection(selection: [number, number]) {
    const textarea = this.textareaRef.current;

    if (textarea) {
      textarea.selectionStart = selection[0];
      textarea.selectionEnd = selection[1];
      return;
    }

    throw new Error('No textarea element reference exists');
  }

  /** wrapper with error handling for props.uploadImage */
  uploadImage(file: File): Promise<Image | Error> {
    const intl = this.props.intl;
    return this.props.uploadImage!(file).catch((e: ApiError | string) => {
      return new Error(
        intl.formatMessage(messages.uploadFileFail, {
          fileName: file.name,
          errorMessage: extractErrorMessageFromResponse(e, this.props.intl),
        })
      );
    });
  }

  /** performs upload process */
  async uploadImages(files: File[]) {
    const intl = this.props.intl;
    if (!this.props.uploadImage) return;
    if (!this.textareaRef.current) return;

    /** Human readable image size limit, i.e 5MB */
    const maxImageSizeString = `${(StaticStore.config.max_image_size / 1024 / 1024).toFixed(2)}MB`;
    /** upload delay to avoid server rate limiter */
    const uploadDelay = 5000;

    this.setState({
      errorLock: true,
      errorMessage: null,
      isErrorShown: false,
      isDisabled: true,
      buttonText: intl.formatMessage(messages.uploading),
    });

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      const isFirst = i === 0;
      const placeholderStart = this.state.text.length === 0 ? '' : '\n';

      const uploadPlaceholder = `${placeholderStart}![${intl.formatMessage(messages.uploadingFile, {
        fileName: file.name,
      })}]()`;
      const uploadPlaceholderLength = uploadPlaceholder.length;
      const selection = this.getSelection();
      /** saved selection in case of error */
      const originalText = this.state.text;
      const restoreSelection = async () => {
        this.setState({
          text: originalText,
        });
        /** sleeping awhile so textarea catch state change and its selection */
        await sleep(100);
        this.setSelection(selection);
      };

      if (file.size > StaticStore.config.max_image_size) {
        this.appendError(
          intl.formatMessage(messages.exceededSize, {
            fileName: file.name,
            maxImageSize: maxImageSizeString,
          })
        );
        continue;
      }

      this.setState({ text: replaceSelection(this.state.text, selection, uploadPlaceholder) }, () => {
        updateJsonItem(LS_SAVED_COMMENT_VALUE, { [this.props.id]: this.state.text });
      });

      !isFirst && (await sleep(uploadDelay));

      const result = await this.uploadImage(file);

      if (result instanceof Error) {
        this.appendError(result.message);
        await restoreSelection();
        continue;
      }

      const markdownString = `${placeholderStart}![${result.name}](${result.url})`;
      this.setState(
        {
          text: replaceSelection(
            this.state.text,
            [selection[0], selection[0] + uploadPlaceholderLength],
            markdownString
          ),
        },
        () => {
          updateJsonItem(LS_SAVED_COMMENT_VALUE, { [this.props.id]: this.state.text });
        }
      );
      /** sleeping awhile so textarea catch state change and its selection */
      await sleep(100);
      const selectionPointer = selection[0] + markdownString.length;
      this.setSelection([selectionPointer, selectionPointer]);
    }

    this.setState({ errorLock: false, isDisabled: false, buttonText: null });
  }

  renderMarkdownTip = () => (
    <div className="comment-form__markdown">
      <FormattedMessage
        id="commentForm.notice-about-styling"
        defaultMessage="Styling with <a>Markdown</a> is supported"
        values={{
          a: (title: string) => (
            <a class="comment-form__markdown-link" target="_blank" href="markdown-help.html">
              {title}
            </a>
          ),
        }}
      />
    </div>
  );

  render() {
    const { theme, mode, simpleView, mix, uploadImage, autofocus, user, intl } = this.props;
    const { isDisabled, isErrorShown, preview, text, buttonText } = this.state;
    const charactersLeft = StaticStore.config.max_comment_size - text.length;
    const errorMessage = this.props.errorMessage || this.state.errorMessage;
    const Labels = {
      main: <FormattedMessage id="commentForm.send" defaultMessage="Send" />,
      edit: <FormattedMessage id="commentForm.save" defaultMessage="Save" />,
      reply: <FormattedMessage id="commentForm.reply" defaultMessage="Reply" />,
    };
    const label = buttonText || Labels[mode || 'main'];
    const placeholderMessage = intl.formatMessage(messages.placeholder);
    return (
      <form
        className={b('comment-form', {
          mods: {
            theme,
            type: mode || 'reply',
            simple: simpleView,
          },
          mix,
        })}
        onSubmit={this.send}
        aria-label={intl.formatMessage(messages.newComment)}
        onDragOver={this.onDragOver}
        onDrop={this.onDrop}
      >
        {!simpleView && (
          <div className="comment-form__control-panel">
            <MarkdownToolbar
              intl={intl}
              allowUpload={Boolean(uploadImage)}
              uploadImages={this.uploadImages}
              textareaId={this.textareaId}
            />
          </div>
        )}
        <div className="comment-form__field-wrapper">
          <TextExpander>
            <TextareaAutosize
              id={this.textareaId}
              ref={this.textareaRef}
              onPaste={this.onPaste}
              className="comment-form__field"
              placeholder={placeholderMessage}
              value={text}
              onInput={this.onInput}
              onKeyDown={this.onKeyDown}
              disabled={isDisabled}
              autofocus={!!autofocus}
              spellcheck={true}
            />
          </TextExpander>
          {charactersLeft < 100 && <span className="comment-form__counter">{charactersLeft}</span>}
        </div>

        {(isErrorShown || !!errorMessage) &&
          (errorMessage || intl.formatMessage(messages.unexpectedError)).split('\n').map((e) => (
            <p className="comment-form__error" role="alert" key={e}>
              {e}
            </p>
          ))}

        <div className="comment-form__actions">
          {user ? (
            <>
              <div>
                {!simpleView && (
                  <Button
                    kind="secondary"
                    theme={theme}
                    size="large"
                    mix="comment-form__button"
                    disabled={isDisabled}
                    onClick={this.getPreview}
                  >
                    <FormattedMessage id="commentForm.preview" defaultMessage="Preview" />
                  </Button>
                )}
                <Button kind="primary" size="large" mix="comment-form__button" type="submit" disabled={isDisabled}>
                  {label}
                </Button>
              </div>

              {!simpleView && mode === 'main' && (
                <div className="comment-form__rss">
                  {this.renderMarkdownTip()}
                  <FormattedMessage id="commentForm.subscribe-by" defaultMessage="Subscribe by" />{' '}
                  <SubscribeByRSS userId={user !== null ? user.id : null} />
                  {StaticStore.config.email_notifications && StaticStore.query.show_email_subscription && (
                    <>
                      {' '}
                      <FormattedMessage id="commentForm.subscribe-or" defaultMessage="or" /> <SubscribeByEmail />
                    </>
                  )}
                </div>
              )}
            </>
          ) : (
            <>
              <Auth />
              {this.renderMarkdownTip()}
            </>
          )}
        </div>

        {
          // TODO: it can be more elegant;
          // for example it can render full comment component here (or above textarea on mobile)
          !!preview && (
            <div className="comment-form__preview-wrapper">
              <div
                className={b('comment-form__preview', {
                  mix: b('raw-content', {}, { theme }),
                })}
                // eslint-disable-next-line react/no-danger
                dangerouslySetInnerHTML={{ __html: preview }}
              />
            </div>
          )
        }
      </form>
    );
  }
}
