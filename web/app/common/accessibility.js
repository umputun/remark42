import { h, Component } from 'preact';

export class A11yButton extends Component {
  constructor(props) {
    super(props);
    this.handleBtnKeyPress = this.handleBtnKeyPress.bind(this);
  }

  handleBtnKeyPress(event) {
    const { onClick } = this.props;
    if (event.key === " " || event.key === "Enter") {
      event.preventDefault();
      onClick();
    }
  }

  render() {
    const { children } = this.props;
    return <children
      role="button"
      tabIndex={0}
      onKeyPress={this.handleBtnKeyPress}
      {...this.props}
    />;
  }
}
