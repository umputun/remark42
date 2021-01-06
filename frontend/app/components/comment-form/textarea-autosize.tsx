import { h, JSX, Component, createRef, RefObject } from 'preact';

export type TextareaAutosizeProps = {
  ref?: RefObject<TextareaAutosize>;
} & Omit<JSX.HTMLAttributes<HTMLTextAreaElement>, 'ref'>;

// TODO: rewrite it to functional component and add ref forwarding
export default class TextareaAutosize extends Component<TextareaAutosizeProps> {
  textareaRef = createRef<HTMLTextAreaElement>();

  componentDidMount() {
    this.autoResize();

    if (this.props.autofocus) this.focus();
  }

  componentDidUpdate(prevProps: TextareaAutosizeProps) {
    if (prevProps.value !== this.props.value) {
      this.autoResize();
    }
  }

  focus(): void {
    setTimeout(() => {
      const { current: textarea } = this.textareaRef;

      if (textarea) {
        textarea.focus();
        textarea.selectionStart = textarea.value.length;
        textarea.selectionEnd = textarea.value.length;
      }
    }, 100);
  }

  /** returns whether selectionStart api supported */
  isSelectionSupported() {
    const { current: textarea } = this.textareaRef;

    if (textarea) {
      return 'selectionStart' in textarea;
    }

    throw new Error('No textarea element reference exists');
  }

  /** returns selection range of a textarea */
  getSelection(): [number, number] {
    const { current: textarea } = this.textareaRef;

    if (textarea) {
      return [textarea.selectionStart, textarea.selectionEnd];
    }

    throw new Error('No textarea element reference exists');
  }

  /** sets selection range of a textarea */
  setSelection(selection: [number, number]) {
    const { current: textarea } = this.textareaRef;

    if (textarea) {
      textarea.selectionStart = selection[0];
      textarea.selectionEnd = selection[1];
      return;
    }

    throw new Error('No textarea element reference exists');
  }

  getValue() {
    const { current: textarea } = this.textareaRef;

    return textarea ? textarea.value : '';
  }

  autoResize() {
    const { current: textarea } = this.textareaRef;

    if (textarea) {
      textarea.style.height = '';
      textarea.style.height = `${textarea.scrollHeight}px`;
    }
  }

  render() {
    return <textarea {...this.props} ref={this.textareaRef} />;
  }
}
