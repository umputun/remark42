import '@ungap/custom-elements';
import '@github/markdown-toolbar-element';
import { h, Component } from 'preact';
import { defineMessages, IntlShape } from 'react-intl';

// TODO: Use SVGR
import BoldIcon from './markdown-toolbar-icons/bold-icon';
import HeaderIcon from './markdown-toolbar-icons/header-icon';
import ItalicIcon from './markdown-toolbar-icons/italic-icon';
import QuoteIcon from './markdown-toolbar-icons/quote-icon';
import CodeIcon from './markdown-toolbar-icons/code-icon';
import LinkIcon from './markdown-toolbar-icons/link-icon';
import ImageIcon from './markdown-toolbar-icons/image-icon';
import UnorderedListIcon from './markdown-toolbar-icons/unordered-list-icon';
import OrderedListIcon from './markdown-toolbar-icons/ordered-list-icon';

interface Props {
  intl: IntlShape;
  textareaId: string;
  uploadImages: (files: File[]) => Promise<void>;
  allowUpload: boolean;
}

interface FileEventTarget extends EventTarget {
  readonly files: FileList | null;
  value: string | null;
}

interface FileInputEvent extends Event {
  readonly currentTarget: FileEventTarget | null;
}

const messages = defineMessages({
  bold: {
    id: 'toolbar.bold',
    defaultMessage: 'Add bold text {shortcut}',
  },
  header: {
    id: 'toolbar.header',
    defaultMessage: 'Add header text',
  },
  italic: {
    id: 'toolbar.italic',
    defaultMessage: 'Add italic text {shortcut}',
  },
  quote: {
    id: 'toolbar.quote',
    defaultMessage: 'Insert a quote',
  },
  code: {
    id: 'toolbar.code',
    defaultMessage: 'Insert a code',
  },
  link: {
    id: 'toolbar.link',
    defaultMessage: 'Add a link {shortcut}',
  },
  unorderedList: {
    id: 'toolbar.unordered-list',
    defaultMessage: 'Add a bulleted list',
  },
  orderedList: {
    id: 'toolbar.ordered-list',
    defaultMessage: 'Add a numbered list',
  },
  attachImage: {
    id: 'toolbar.attach-image',
    defaultMessage: 'Attach the image, drag & drop or paste from clipboard',
  },
});

export default class MarkdownToolbar extends Component<Props> {
  constructor(props: Props) {
    super(props);
    this.uploadImages = this.uploadImages.bind(this);
  }
  async uploadImages(e: Event) {
    const currentTarget = (e as FileInputEvent).currentTarget;
    if (!(this.props.allowUpload && currentTarget && currentTarget.files && currentTarget.files.length !== 0)) return;
    const files = Array.from(currentTarget.files);
    await this.props.uploadImages(files);
    currentTarget.value = null;
  }
  render(props: Props) {
    const intl = props.intl;

    const boldLabel = intl.formatMessage(messages.bold, { shortcut: '<cmd-b>' });
    const headerLabel = intl.formatMessage(messages.header);
    const italicLabel = intl.formatMessage(messages.italic, { shortcut: '<cmd-i>' });
    const quoteLabel = intl.formatMessage(messages.quote);
    const codeLabel = intl.formatMessage(messages.code);
    const linkLabel = intl.formatMessage(messages.link, { shortcut: '<cmd-k>' });
    const unorderedListLabel = intl.formatMessage(messages.unorderedList);
    const orderedListLabel = intl.formatMessage(messages.orderedList);
    const attachImageLabel = intl.formatMessage(messages.attachImage);

    return (
      <markdown-toolbar className="comment-form__toolbar" for={props.textareaId}>
        <div className="comment-form__toolbar-group">
          <md-header className="comment-form__toolbar-item" title={headerLabel} aria-label={headerLabel}>
            <HeaderIcon />
          </md-header>
          <md-bold className="comment-form__toolbar-item" title={boldLabel} aria-label={boldLabel}>
            <BoldIcon />
          </md-bold>
          <md-italic className="comment-form__toolbar-item" title={italicLabel} aria-label={italicLabel}>
            <ItalicIcon />
          </md-italic>
        </div>
        <div className="comment-form__toolbar-group">
          <md-quote className="comment-form__toolbar-item" title={quoteLabel} aria-label={quoteLabel}>
            <QuoteIcon />
          </md-quote>
          <md-code className="comment-form__toolbar-item" title={codeLabel} aria-label={codeLabel}>
            <CodeIcon />
          </md-code>
          <md-link className="comment-form__toolbar-item" title={linkLabel} aria-label={linkLabel}>
            <LinkIcon />
          </md-link>
          {this.props.allowUpload ? (
            <label className="comment-form__toolbar-item" title={attachImageLabel} aria-label={attachImageLabel}>
              <input multiple className="comment-form__toolbar-file-input" type="file" onChange={this.uploadImages} />
              <ImageIcon />
            </label>
          ) : null}
        </div>
        <div className="comment-form__toolbar-group">
          <md-unordered-list
            className="comment-form__toolbar-item"
            title={unorderedListLabel}
            aria-label={unorderedListLabel}
          >
            <UnorderedListIcon />
          </md-unordered-list>
          <md-ordered-list
            className="comment-form__toolbar-item"
            title={orderedListLabel}
            aria-label={orderedListLabel}
          >
            <OrderedListIcon />
          </md-ordered-list>
        </div>
      </markdown-toolbar>
    );
  }
}
