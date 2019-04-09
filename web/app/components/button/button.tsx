/** @jsx h */
import { Component, h, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import noop from '@app/utils/noop';
import { Theme } from '@app/common/types';

interface Props {
  type?: string;
  kind?: string;
  theme: Theme;
  mix?: string;

  onClick?: (e: MouseEvent) => void;
  onFocus?: (e: FocusEvent) => void;
  onBlur?: (e: FocusEvent) => void;
}

interface State {
  isClicked: boolean;
  isFocused: boolean;
}

export class Button extends Component<JSX.HTMLAttributes & Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      isClicked: false,
      isFocused: false,
    };

    this.onMouseDown = this.onMouseDown.bind(this);
    this.onFocus = this.onFocus.bind(this);
    this.onBlur = this.onBlur.bind(this);
  }

  onMouseDown() {
    this.setState({
      isClicked: true,
    });
  }

  onClick(e: MouseEvent) {
    this.props.onClick!(e);
  }

  onBlur(e: FocusEvent) {
    this.setState({
      isClicked: false,
      isFocused: false,
    });

    this.props.onBlur!(e);
  }

  onFocus(e: FocusEvent) {
    this.setState({
      isFocused: true,
    });

    this.props.onFocus!(e);
  }

  render(props: RenderableProps<JSX.HTMLAttributes & Props>, state: State) {
    const { children, className } = props;
    const { isClicked, isFocused } = state;

    let rclassName = b(
      'button',
      { mix: props.mix },
      { theme: props.theme, type: props.type, kind: props.kind, clicked: isClicked, focused: isFocused }
    );
    if (className) {
      rclassName +=
        ' ' +
        b(
          className,
          {},
          { theme: props.theme, type: props.type, kind: props.kind, clicked: isClicked, focused: isFocused }
        );
    }

    const localProps = { ...props };
    delete localProps.children;
    delete localProps.mix;

    return (
      <button
        {...localProps}
        className={rclassName}
        onMouseDown={this.onMouseDown}
        onBlur={this.onBlur}
        onFocus={this.onFocus}
      >
        {children}
      </button>
    );
  }
}

Button.defaultProps = {
  type: 'button',
  onClick: noop,
  onBlur: noop,
  onFocus: noop,
};
