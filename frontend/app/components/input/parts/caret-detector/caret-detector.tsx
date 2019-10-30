/** @jsx h */
import { Component, h } from 'preact';

interface Props {
  caretPosition: number;
  text?: string;
}

interface State {
  caretPosition: number;
  text?: string;
}

export default class CaretDetector extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = { ...props };

    this.getContent = this.getContent.bind(this);
  }

  getContent() {
    const { text, caretPosition } = this.props;
    const textStart = text && text.substring(0, caretPosition + 1);
    const textEnd = text && text.substring(caretPosition + 1);

    return {
      textStart,
      textEnd,
    };
  }

  render() {
    const { textStart, textEnd } = this.getContent();

    return (
      <div>
        {textStart}
        <span>q</span>
        {textEnd}
      </div>
    );
  }
}
