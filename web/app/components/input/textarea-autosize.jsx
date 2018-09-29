/** @jsx h */
import { h, Component } from 'preact';

export default class TextareaAutosize extends Component {
  constructor(props) {
    super(props);

    this.onRef = this.onRef.bind(this);
  }
  componentDidMount() {
    this.autoResize();
  }
  componentDidUpdate(prevProps) {
    if (prevProps.value !== this.props.value) {
      this.autoResize();
    }
  }
  onRef(node) {
    this.textareaRef = node;
  }
  autoResize() {
    this.textareaRef.style.height = '';
    this.textareaRef.style.height = `${this.textareaRef.scrollHeight}px`;
  }
  render(props) {
    return (
      // We set text as a child of textarea and not in value property for a reason.
      // It's a workaround for the bug described here https://github.com/developit/preact/issues/326
      <textarea {...props} ref={this.onRef}>
        {props.value}
      </textarea>
    );
  }
}
