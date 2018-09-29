/** @jsx h */
import { Component, h } from 'preact';
import noop from '../../utils/noop';

export default class Button extends Component {
  constructor(props) {
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

  onClick(e) {
    this.props.onClick(e);
  }

  onBlur(e) {
    this.setState({
      isClicked: false,
      isFocused: false,
    });

    this.props.onBlur(e);
  }

  onFocus(e) {
    this.setState({
      isFocused: true,
    });

    this.props.onFocus(e);
  }

  render(props, state) {
    const { children, mix, mods, ...rest } = props;
    const { isClicked, isFocused } = state;

    return (
      <button
        {...rest}
        className={b('button', { mix, mods }, { clicked: isClicked, focused: isFocused })}
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
