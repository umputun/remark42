/** @jsx h */
import { h, Component, RenderableProps } from 'preact';
import '@github/markdown-toolbar-element';
import BoldIcon from './markdown-toolbar-icons/bold-icon';
import HeaderIcon from './markdown-toolbar-icons/header-icon';
import ItalicIcon from './markdown-toolbar-icons/italic-icon';
import QuoteIcon from './markdown-toolbar-icons/quote-icon';
import CodeIcon from './markdown-toolbar-icons/code-icon';
import LinkIcon from './markdown-toolbar-icons/link-icon';

interface Props {
  textareaId: string;
}

const boldLabel = 'Add bold text <cmd-b>';
const headerLabel = 'Add header text';
const italicLabel = 'Add italic text <cmd-i>';
const quoteLabel = 'Insert a quote';
const codeLabel = 'Insert a code';
const linkLabel = 'Add a link <cmd-k>';

export default class MarkdownToolbar extends Component<Props> {
  render(props: RenderableProps<Props>) {
    return (
      <markdown-toolbar className="input__toolbar" for={props.textareaId}>
        <div className="input__toolbar-group">
          <md-header className="input__toolbar-item" title={headerLabel} aria-label={headerLabel}>
            <HeaderIcon />
          </md-header>
          <md-bold className="input__toolbar-item" title={boldLabel} aria-label={boldLabel}>
            <BoldIcon />
          </md-bold>
          <md-italic className="input__toolbar-item" title={italicLabel} aria-label={italicLabel}>
            <ItalicIcon />
          </md-italic>
        </div>
        <div className="input__toolbar-group">
          <md-quote className="input__toolbar-item" title={quoteLabel} aria-label={quoteLabel}>
            <QuoteIcon />
          </md-quote>
          <md-code className="input__toolbar-item" title={codeLabel} aria-label={codeLabel}>
            <CodeIcon />
          </md-code>
          <md-link className="input__toolbar-item" title={linkLabel} aria-label={linkLabel}>
            <LinkIcon />
          </md-link>
        </div>
      </markdown-toolbar>
    );
  }
}
