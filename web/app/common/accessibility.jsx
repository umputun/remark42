import { h, cloneElement, Component } from 'preact';

export class A11yButton extends Component {
  constructor(props) {
    super(props);
    this.handleBtnKeyPress = this.handleBtnKeyPress.bind(this);
  }

  handleBtnKeyPress(event) {
    const { onClick } = this.props;
    if (event.key === " " || event.key === "Enter") {
      event.preventDefault();
      onClick && onClick();
    }
  }

  render() {
    const childrenWithExtraProp = this.props.children.map(child =>
      cloneElement(child, {
        role:"button",
        tabIndex: 0,
        onKeyPress:this.handleBtnKeyPress,
        ...this.props
      })
    )[0];
    
    return childrenWithExtraProp;
  }
}
