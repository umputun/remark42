/** @jsx h */
import { Component, h, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { Button } from '@app/components/button';
import { Theme } from '@app/common/types';

interface Props {
  title: string;
  heading?: string;
  isActive?: boolean;
  onTitleClick?: () => void;
  mix?: string;
  theme: Theme;
}

interface State {
  isActive: boolean;
}

export default class Dropdown extends Component<Props, State> {
  rootNode?: HTMLDivElement;

  constructor(props: Props) {
    super(props);

    this.state = {
      isActive: props.isActive || false,
    };
  }

  onTitleClick() {
    this.setState({
      isActive: !this.state.isActive,
    });

    if (this.props.onTitleClick) {
      this.props.onTitleClick();
    }
  }

  receiveMessage(e: { data: string | object }) {
    try {
      const data = typeof e.data === 'string' ? JSON.parse(e.data) : e.data;

      if (data.clickOutside) {
        if (this.state.isActive) {
          this.setState({
            isActive: false,
          });
        }
      }
    } catch (e) {}
  }

  onOutsideClick(e: MouseEvent) {
    if (this.rootNode && !this.rootNode.contains(e.target as Node)) {
      if (this.state.isActive) {
        this.setState({
          isActive: false,
        });
      }
    }
  }

  componentDidMount() {
    document.addEventListener('click', e => this.onOutsideClick(e));

    window.addEventListener('message', e => this.receiveMessage(e));
  }

  componentWillUnmount() {
    document.removeEventListener('click', e => this.onOutsideClick(e));

    window.removeEventListener('message', e => this.receiveMessage(e));
  }

  render(props: RenderableProps<Props>, { isActive }: State) {
    const { title, heading, children, mix } = props;

    return (
      <div className={b('dropdown', { mix }, { theme: props.theme, active: isActive })} ref={r => (this.rootNode = r)}>
        <Button
          aria-haspopup="listbox"
          aria-expanded={isActive && 'true'}
          mix="dropdown__title"
          type="button"
          onClick={() => this.onTitleClick()}
          theme="light"
        >
          {title}
        </Button>

        <div className="dropdown__content" tabIndex={-1} role="listbox">
          {heading && <div className="dropdown__heading">{heading}</div>}
          <div className="dropdown__items">{children}</div>
        </div>
      </div>
    );
  }
}
