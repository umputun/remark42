/** @jsx createElement */
import { createElement, Component, createRef, Fragment } from 'preact';
import { FormattedMessage, IntlShape, defineMessages } from 'react-intl';
import b, { Mix } from 'bem-react-helper';

import { User, Theme, Image, ApiError } from '@app/common/types';
import { StaticStore } from '@app/common/static_store';
import { pageTitle } from '@app/common/settings';
import { extractErrorMessageFromResponse } from '@app/utils/errorUtils';
import { isUserAnonymous } from '@app/utils/isUserAnonymous';
import { sleep } from '@app/utils/sleep';
import { replaceSelection } from '@app/utils/replaceSelection';
import { Button } from '@app/components/button';
import Auth from '@app/components/auth';
import { getJsonItem, updateJsonItem } from '@app/common/local-storage';
import { LS_SAVED_COMMENT_VALUE } from '@app/common/constants';

import { SubscribeByEmail } from './__subscribe-by-email';
import { SubscribeByRSS } from './__subscribe-by-rss';

import MarkdownToolbar from './markdown-toolbar';
import TextareaAutosize from './textarea-autosize';
import { TextExpander } from './text-expander';

let textareaId = 0;

export interface Props {
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
}

export interface State {
  preview: string | null;
  isErrorShown: boolean;
  /** error message, if contains newlines, it will be splitted to multiple errors */
  errorMessage: string | null;
  /** prevents error hiding on input event */
  errorLock: boolean;
  isDisabled: boolean;
  maxLength: number;
  /** main input value */
  text: string;
  /** override main button text */
  buttonText: null | string;
}

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

export class CommentForm extends Component<Props, State> {
  /** reference to textarea element */
  textAreaRef = createRef<TextareaAutosize>();
  textareaId: string;

  constructor(props: Props) {
    super(props);
    textareaId = textareaId + 1;
    this.textareaId = `textarea_${textareaId}`;

    const savedComments = getJsonItem(LS_SAVED_COMMENT_VALUE);
    let text = '';

    if (savedComments !== null && savedComments[props.id]) {
      text = savedComments[props.id];
    }

    if (props.value) {
      text = props.value;
    }

    this.state = {
      preview: null,
      isErrorShown: false,
      errorMessage: null,
      errorLock: false,
      isDisabled: false,
      maxLength: StaticStore.config.max_comment_size,
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

  componentWillReceiveProps(nextProps: Props) {
    if (nextProps.value !== this.props.value) {
      this.setState({ text: nextProps.value || '' });
      this.props.autofocus && this.textAreaRef.current && this.textAreaRef.current.focus();
    }
    if (nextProps.user && !this.props.value) {
      this.setState({
        isErrorShown: false,
        errorMessage: null,
      });
    }
  }

  shouldComponentUpdate(nextProps: Props, nextState: State) {
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

    updateJsonItem(LS_SAVED_COMMENT_VALUE, { [this.props.id]: value });

    if (this.state.errorLock) {
      this.setState({
        preview: null,
        text: value,
      });
      return;
    }

    this.setState({
      isErrorShown: false,
      errorMessage: null,
      preview: null,
      text: value,
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
      updateJsonItem<Record<string, string>>(LS_SAVED_COMMENT_VALUE, data => {
        delete data[this.props.id];

        return data;
      });
      this.setState({ preview: null, text: '' });
    } catch (e) {
      this.setState({
        isErrorShown: true,
        errorMessage: extractErrorMessageFromResponse(e, this.props.intl),
      });
    }

    this.setState({ isDisabled: false });
  };

  getPreview() {
    const text = this.textAreaRef.current ? this.textAreaRef.current.getValue() : this.state.text;

    if (!text || !text.trim()) return;

    this.setState({ isErrorShown: false, errorMessage: null, text });

    this.props
      .getPreview(text)
      .then(preview => this.setState({ preview }))
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
      errorMessage: this.state.errorMessage + '\n' + errors.join('\n'),
      isErrorShown: true,
    });
  }

  onDragOver(e: DragEvent) {
    if (!this.props.user) e.preventDefault();
    if (!this.props.uploadImage) return;
    if (StaticStore.config.max_image_size === 0) return;
    if (!this.textAreaRef) return;
    if (!e.dataTransfer) return;
    const items = Array.from(e.dataTransfer.items);
    if (Array.from(items).filter(i => i.kind === 'file' && ImageMimeRegex.test(i.type)).length === 0) return;
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

    const data = Array.from(e.dataTransfer.files).filter(f => ImageMimeRegex.test(f.type));
    if (data.length === 0) return;

    e.preventDefault();

    this.uploadImages(data);
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
    if (!this.textAreaRef.current) return;

    /** Human readable image size limit, i.e 5MB */
    const maxImageSizeString = (StaticStore.config.max_image_size / 1024 / 1024).toFixed(2) + 'MB';
    /** upload delay to avoid server rate limiter */
    const uploadDelay = 5000;

    const isSelectionSupported = this.textAreaRef.current.isSelectionSupported();

    this.setState({
      errorLock: true,
      errorMessage: null,
      isErrorShown: false,
      isDisabled: true,
      buttonText: intl.formatMessage(messages.uploading),
    });

    // TODO: remove legacy code, now we don't support IE
    // fallback for ie < 9
    if (!isSelectionSupported) {
      for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const isFirst = i === 0;
        const placeholderStart = this.state.text.length === 0 ? '' : '\n';

        if (file.size > StaticStore.config.max_image_size) {
          this.appendError(
            intl.formatMessage(messages.exceededSize, {
              fileName: file.name,
              maxImageSize: maxImageSizeString,
            })
          );
          continue;
        }

        !isFirst && (await sleep(uploadDelay));

        const result = await this.uploadImage(file);

        if (result instanceof Error) {
          this.appendError(result.message);
          continue;
        }

        const markdownString = `${placeholderStart}![${result.name}](${result.url})`;
        this.setState({
          text: this.state.text + markdownString,
        });
      }

      this.setState({ errorLock: false, isDisabled: false, buttonText: null });
      return;
    }

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      const isFirst = i === 0;
      const placeholderStart = this.state.text.length === 0 ? '' : '\n';

      const uploadPlaceholder = `${placeholderStart}![${intl.formatMessage(messages.uploadingFile, {
        fileName: file.name,
      })}]()`;
      const uploadPlaceholderLength = uploadPlaceholder.length;
      const selection = this.textAreaRef.current.getSelection();
      /** saved selection in case of error */
      const originalText = this.state.text;
      const restoreSelection = async () => {
        this.setState({
          text: originalText,
        });
        /** sleeping awhile so textarea catch state change and its selection */
        await sleep(100);
        this.textAreaRef.current!.setSelection(selection);
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

      this.setState({
        text: replaceSelection(this.state.text, selection, uploadPlaceholder),
      });

      !isFirst && (await sleep(uploadDelay));

      const result = await this.uploadImage(file);

      if (result instanceof Error) {
        this.appendError(result.message);
        await restoreSelection();
        continue;
      }

      const markdownString = `${placeholderStart}![${result.name}](${result.url})`;
      this.setState({
        text: replaceSelection(this.state.text, [selection[0], selection[0] + uploadPlaceholderLength], markdownString),
      });
      /** sleeping awhile so textarea catch state change and its selection */
      await sleep(100);
      const selectionPointer = selection[0] + markdownString.length;
      this.textAreaRef.current.setSelection([selectionPointer, selectionPointer]);
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

  render(props: Props, { isDisabled, isErrorShown, errorMessage, preview, maxLength, text, buttonText }: State) {
    const charactersLeft = maxLength - text.length;
    errorMessage = props.errorMessage || errorMessage;
    const Labels = {
      main: <FormattedMessage id="commentForm.send" defaultMessage="Send" />,
      edit: <FormattedMessage id="commentForm.save" defaultMessage="Save" />,
      reply: <FormattedMessage id="commentForm.reply" defaultMessage="Reply" />,
    };
    const label = buttonText || Labels[props.mode || 'main'];
    const intl = this.props.intl;
    const placeholderMessage = intl.formatMessage(messages.placeholder);
    return (
      <form
        className={b('comment-form', {
          mods: {
            theme: props.theme,
            type: props.mode || 'reply',
            simple: props.simpleView,
          },
          mix: props.mix,
        })}
        onSubmit={this.send}
        aria-label={intl.formatMessage(messages.newComment)}
        onDragOver={this.onDragOver}
        onDrop={this.onDrop}
      >
        {!props.simpleView && (
          <div className="comment-form__control-panel">
            <MarkdownToolbar
              intl={intl}
              allowUpload={Boolean(this.props.uploadImage)}
              uploadImages={this.uploadImages}
              textareaId={this.textareaId}
            />
          </div>
        )}
        <div className="comment-form__field-wrapper">
          <TextExpander>
            <TextareaAutosize
              id={this.textareaId}
              onPaste={this.onPaste}
              ref={this.textAreaRef}
              className="comment-form__field"
              placeholder={placeholderMessage}
              value={text}
              maxLength={maxLength}
              onInput={this.onInput}
              onKeyDown={this.onKeyDown}
              disabled={isDisabled}
              autofocus={!!props.autofocus}
              spellcheck={true}
            />
          </TextExpander>
          {charactersLeft < 100 && <span className="comment-form__counter">{charactersLeft}</span>}
        </div>

        {(isErrorShown || !!errorMessage) &&
          (errorMessage || intl.formatMessage(messages.unexpectedError)).split('\n').map(e => (
            <p className="comment-form__error" role="alert" key={e}>
              {e}
            </p>
          ))}

        <div className="comment-form__actions">
          {this.props.user ? (
            <Fragment>
              <div>
                {!props.simpleView && (
                  <Button
                    kind="secondary"
                    theme={props.theme}
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

              {!props.simpleView && props.mode === 'main' && (
                <div className="comment-form__rss">
                  {this.renderMarkdownTip()}
                  <FormattedMessage id="commentForm.subscribe-by" defaultMessage="Subscribe by" />{' '}
                  <SubscribeByRSS userId={props.user !== null ? props.user.id : null} />
                  {StaticStore.config.email_notifications && (
                    <Fragment>
                      {' '}
                      <FormattedMessage id="commentForm.subscribe-or" defaultMessage="or" /> <SubscribeByEmail />
                    </Fragment>
                  )}
                </div>
              )}
            </Fragment>
          ) : (
            <Fragment>
              <Auth />
              {this.renderMarkdownTip()}
            </Fragment>
          )}
        </div>

        {// TODO: it can be more elegant;
        // for example it can render full comment component here (or above textarea on mobile)
        !!preview && (
          <div className="comment-form__preview-wrapper">
            <div
              className={b('comment-form__preview', {
                mix: b('raw-content', {}, { theme: props.theme }),
              })}
              dangerouslySetInnerHTML={{ __html: preview }}
            />
          </div>
        )}
      </form>
    );
  }
}
