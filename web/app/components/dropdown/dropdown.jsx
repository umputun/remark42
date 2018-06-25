/** @jsx h */
import { Component, h } from 'preact';

import Button from 'components/button';

export default class Dropdown extends Component {
  constructor(props) {
    super(props);

    this.state = {
      isActive: props.isActive || false,
    };

    this.onTitleClick = this.onTitleClick.bind(this);
    this.onOutsideClick = this.onOutsideClick.bind(this);
  }

  onTitleClick() {
    this.setState({
      isActive: !this.state.isActive,
    });

    if (this.props.onTitleClick) {
      this.props.onTitleClick();
    }
  }

  onOutsideClick(e) {
    if (!this.rootNode.contains(e.target)) {
      if (this.state.isActive) {
        this.setState({
          isActive: false,
        });
      }
    }
  }

  componentDidMount() {
    document.addEventListener('click', this.onOutsideClick);

    if (parent) {
      parent.document.addEventListener('click', this.onOutsideClick);
    }
  }

  componentWillUnmount() {
    document.removeEventListener('click', this.onOutsideClick);

    if (parent) {
      parent.document.removeEventListener('click', this.onOutsideClick);
    }
  }

  render(props, { isActive }) {
    const { title, heading, children, mix, mods } = props;

    return (
      <div className={b('dropdown', { mix, mods }, { active: isActive })} ref={r => (this.rootNode = r)}>
        <Button
          aria-haspopup="listbox"
          aria-expanded={isActive && 'true'}
          mix="dropdown__title"
          type="button"
          onClick={this.onTitleClick}
        >
          {title}
        </Button>

        <div className="dropdown__content" tabindex="-1" role="listbox">
          {heading && <div className="dropdown__heading">{heading}</div>}
          <div className="dropdown__items">{children}</div>
        </div>
      </div>
    );
  }
}
