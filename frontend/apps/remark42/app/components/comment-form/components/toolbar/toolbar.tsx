import '@ungap/custom-elements';
import '@github/markdown-toolbar-element';
import { h } from 'preact';
import { defineMessages, useIntl } from 'react-intl';
import clsx from 'clsx';

// TODO: Use SVGR
import { BoldIcon } from './icons/bold-icon';
import { HeaderIcon } from './icons/header-icon';
import { ItalicIcon } from './icons/italic-icon';
import { QuoteIcon } from './icons/quote-icon';
import { CodeIcon } from './icons/code-icon';
import { LinkIcon } from './icons/link-icon';
import { ImageIcon } from './icons/image-icon';
import { UnorderedListIcon } from './icons/unordered-list-icon';
import { OrderedListIcon } from './icons/ordered-list-icon';

import styles from './toolbar.module.css';

export function MarkdownToolbar({
  textareaId,
  uploadImages,
}: {
  textareaId: string;
  uploadImages?(files: File[]): Promise<void>;
}) {
  const intl = useIntl();

  function handleUploadImages({ currentTarget }: preact.JSX.TargetedEvent<HTMLInputElement>) {
    const files = Array.from(currentTarget.files ?? []);
    uploadImages?.(files).finally(() => {
      currentTarget.value = '';
    });
  }

  return (
    <markdown-toolbar className={clsx('comment-form-toolbar', styles.root)} for={textareaId}>
      <div>
        <md-header
          className={clsx('comment-form-toolbar-item', styles.item)}
          title={intl.formatMessage(messages.header)}
        >
          <HeaderIcon />
        </md-header>
        <md-bold
          className={clsx('comment-form-toolbar-item', styles.item)}
          title={intl.formatMessage(messages.bold, { shortcut: '<cmd-b>' })}
        >
          <BoldIcon />
        </md-bold>
        <md-italic
          className={clsx('comment-form-toolbar-item', styles.item)}
          title={intl.formatMessage(messages.italic, { shortcut: '<cmd-i>' })}
        >
          <ItalicIcon />
        </md-italic>
      </div>
      <div>
        <md-quote className={clsx('comment-form-toolbar-item', styles.item)} title={intl.formatMessage(messages.quote)}>
          <QuoteIcon />
        </md-quote>
        <md-code className={clsx('comment-form-toolbar-item', styles.item)} title={intl.formatMessage(messages.code)}>
          <CodeIcon />
        </md-code>
        <md-link
          className={clsx('comment-form-toolbar-item', styles.item)}
          title={intl.formatMessage(messages.link, { shortcut: '<cmd-k>' })}
        >
          <LinkIcon />
        </md-link>
        {uploadImages ? (
          <label
            className={clsx('comment-form-toolbar-item', styles.item)}
            title={intl.formatMessage(messages.attachImage)}
          >
            <input multiple className={styles.fileInput} type="file" onChange={handleUploadImages} />
            <ImageIcon />
          </label>
        ) : null}
      </div>
      <div>
        <md-unordered-list
          className={clsx('comment-form-toolbar-item', styles.item)}
          title={intl.formatMessage(messages.unorderedList)}
        >
          <UnorderedListIcon />
        </md-unordered-list>
        <md-ordered-list
          className={clsx('comment-form-toolbar-item', styles.item)}
          title={intl.formatMessage(messages.orderedList)}
        >
          <OrderedListIcon />
        </md-ordered-list>
      </div>
    </markdown-toolbar>
  );
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
