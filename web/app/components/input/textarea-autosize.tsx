/** @jsx h */
import { h, Component, RenderableProps } from 'preact';

type Props = JSX.HTMLAttributes & {
  autofocus: boolean;
};

export default class TextareaAutosize extends Component<Props> {
  textareaRef?: HTMLTextAreaElement;

  constructor(props: Props) {
    super(props);

    this.onRef = this.onRef.bind(this);
  }
  componentDidMount() {
    this.autoResize();

    if (this.props.autofocus) this.focus();
  }
  componentDidUpdate(prevProps: Props) {
    if (prevProps.value !== this.props.value) {
      this.autoResize();
    }
  }

  focus(): void {
    setTimeout(() => {
      if (this.textareaRef) {
        this.textareaRef.focus();
        this.textareaRef.selectionStart = this.textareaRef.selectionEnd = this.textareaRef.value.length;
      }
    }, 100);
  }

  onRef(node: HTMLTextAreaElement) {
    this.textareaRef = node;
  }
  autoResize() {
    if (this.textareaRef) {
      this.textareaRef.style.height = '';
      this.textareaRef.style.height = `${this.textareaRef.scrollHeight}px`;
    }
  }
  render(props: RenderableProps<Props>) {
    return (
      // We set text as a child of textarea and not in value property for a reason.
      // It's a workaround for the bug described here https://github.com/developit/preact/issues/326
      <textarea {...props} ref={this.onRef}>
        {props.value}
      </textarea>
    );
  }
}
