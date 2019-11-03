/** @jsx h */
import { Component, h } from 'preact';

// import './caret-detector.scss';

import b from 'bem-react-helper';

interface Props {
  caretPosition: number;
  text?: string;
}

interface State {
  caretPosition: number;
  text?: string;
}

export default class CaretDetector extends Component<Props, State> {
  span?: HTMLSpanElement;

  constructor(props: Props) {
    super(props);

    this.state = { ...props };

    this.getContent = this.getContent.bind(this);
    this.getCaretPosition = this.getCaretPosition.bind(this);
    this.setText = this.setText.bind(this);
  }

  getCaretPosition() {
    if (!this.span)
      return {
        top: undefined,
        left: undefined,
      };

    const span = this.span;

    const top = span.offsetTop;
    const left = span.offsetLeft;

    return {
      top,
      left,
    };
  }

  setText(text: string) {
    this.setState({
      text,
    });
  }

  getContent() {
    const { text, caretPosition } = this.props;
    const textStart = text && text.substring(0, caretPosition);
    const textEnd = text && text.substring(caretPosition);

    return {
      textStart,
      textEnd,
    };
  }

  render() {
    const { textStart, textEnd } = this.getContent();

    const className = b('caret-detector');

    return (
      <div className={className}>
        {textStart}
        <span ref={ref => (this.span = ref)}>q</span>
        {textEnd}
      </div>
    );
  }
}
