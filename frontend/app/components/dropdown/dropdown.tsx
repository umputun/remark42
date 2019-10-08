/** @jsx h */
import { Component, h, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { Button } from '@app/components/button';
import { Theme } from '@app/common/types';
import { sleep } from '@app/utils/sleep';
import { isUndefined } from 'lodash';
import { DropdownItem } from '@app/components/dropdown/index';

interface Props {
  title: string;
  titleClass?: string;
  heading?: string;
  isActive?: boolean;
  onTitleClick?: () => void;
  mix?: string;
  theme: Theme;
  onOpen?: (root: HTMLDivElement) => {};
  onClose?: (root: HTMLDivElement) => {};
  emojiList?: string[];
  activeListEl?: number;
}

interface State {
  isActive: boolean;
  contentTranslateX: number;
  activeListEl: number;
}

export default class Dropdown extends Component<Props, State> {
  rootNode?: HTMLDivElement;

  constructor(props: Props) {
    super(props);

    let { activeListEl } = this.props;
    if (!activeListEl) {
      activeListEl = 0;
    }

    this.state = {
      isActive: props.isActive || false,
      contentTranslateX: 0,
      activeListEl,
    };

    this.onOutsideClick = this.onOutsideClick.bind(this);
    this.receiveMessage = this.receiveMessage.bind(this);
    this.updateState = this.updateState.bind(this);
    this.__onOpen = this.__onOpen.bind(this);
    this.__onClose = this.__onClose.bind(this);
  }

  generateList(list: string[]) {
    return list.map((emoji, index) => <DropdownItem active={index === this.props.activeListEl}>{emoji}</DropdownItem>);
  }

  updateState(props: Props, isActive?: boolean) {
    if (isUndefined(props.isActive) || props.isActive === this.state.isActive) {
      return;
    }

    this.setState({
      isActive: isActive ? !isActive : !props.isActive,
    });

    this.onTitleClick();
  }

  onTitleClick() {
    const isActive = !this.state.isActive;
    const contentTranslateX = isActive ? this.state.contentTranslateX : 0;
    this.setState(
      {
        contentTranslateX,
        isActive,
      },
      async () => {
        await this.__adjustDropDownContent();
        if (isActive) {
          this.__onOpen();
          this.props.onOpen && this.props.onOpen(this.rootNode!);
        } else {
          this.__onClose();
          this.props.onClose && this.props.onClose(this.rootNode!);
        }

        if (this.props.onTitleClick) {
          this.props.onTitleClick();
        }
      }
    );
  }

  storedDocumentHeight: string | null = null;
  storedDocumentHeightSet: boolean = false;
  checkInterval: number | undefined = undefined;

  __onOpen() {
    const isChildOfDropDown = (() => {
      if (!this.rootNode) return false;
      let parent = this.rootNode.parentElement!;
      while (parent !== document.body) {
        if (parent.classList.contains('dropdown')) return true;
        parent = parent.parentElement!;
      }
      return false;
    })();
    if (isChildOfDropDown) return;

    this.storedDocumentHeight = document.body.style.minHeight;
    this.storedDocumentHeightSet = true;

    let prevDcBottom: number | null = null;

    this.checkInterval = window.setInterval(() => {
      if (!this.rootNode || !this.state.isActive) return;
      const windowHeight = window.innerHeight;
      const dcBottom = (() => {
        const dc = Array.from(this.rootNode.children).find(c => c.classList.contains('dropdown__content'));
        if (!dc) return 0;
        const rect = dc.getBoundingClientRect();
        return window.scrollY + Math.abs(rect.top) + dc.scrollHeight + 10;
      })();
      if (prevDcBottom === null && dcBottom <= windowHeight) return;
      if (dcBottom !== prevDcBottom) {
        prevDcBottom = dcBottom;
        document.body.style.minHeight = dcBottom + 'px';
      }
    }, 100);
  }

  __onClose() {
    window.clearInterval(this.checkInterval);
    if (this.storedDocumentHeightSet) {
      document.body.style.minHeight = this.storedDocumentHeight;
    }
  }

  async __adjustDropDownContent() {
    if (!this.rootNode) return;
    const dc = this.rootNode.querySelector<HTMLDivElement>('.dropdown__content');
    if (!dc) return;
    await sleep(10);
    const rect = dc.getBoundingClientRect();
    if (rect.left > 0) {
      const wWindow = window.innerWidth;
      if (rect.right <= wWindow) return;
      const delta = rect.right - wWindow;
      const max = Math.min(rect.left, delta);
      this.setState({
        contentTranslateX: -max,
      });
      return;
    }
    this.setState({
      contentTranslateX: -rect.left,
    });
  }

  receiveMessage(e: { data: string | object }) {
    try {
      const data = typeof e.data === 'string' ? JSON.parse(e.data) : e.data;

      if (!data.clickOutside) return;
      if (!this.state.isActive) return;
      this.setState(
        {
          contentTranslateX: 0,
          isActive: false,
        },
        () => {
          this.__onClose();
          this.props.onClose && this.props.onClose(this.rootNode!);
        }
      );
    } catch (e) {}
  }

  onOutsideClick(e: MouseEvent) {
    if (!this.rootNode || this.rootNode.contains(e.target as Node) || !this.state.isActive) return;
    this.setState(
      {
        contentTranslateX: 0,
        isActive: false,
      },
      () => {
        this.__onClose();
        this.props.onClose && this.props.onClose(this.rootNode!);
      }
    );
  }

  componentDidMount() {
    document.addEventListener('click', this.onOutsideClick);

    window.addEventListener('message', this.receiveMessage);
  }

  componentWillUnmount() {
    document.removeEventListener('click', this.onOutsideClick);

    window.removeEventListener('message', this.receiveMessage);
  }

  render(props: RenderableProps<Props>, { isActive }: State) {
    let { children } = props;
    const { title, titleClass, heading, mix } = props;
    this.updateState(props);

    {
      if (this.props.emojiList) {
        children = this.generateList(this.props.emojiList);
      }
    }

    return (
      <div className={b('dropdown', { mix }, { theme: props.theme, active: isActive })} ref={r => (this.rootNode = r)}>
        <Button
          aria-haspopup="listbox"
          aria-expanded={isActive && 'true'}
          mix="dropdown__title"
          type="button"
          onClick={() => this.onTitleClick()}
          theme="light"
          className={titleClass}
        >
          {title}
        </Button>

        <div
          className="dropdown__content"
          tabIndex={-1}
          role="listbox"
          style={{ transform: `translateX(${this.state.contentTranslateX}px)` }}
        >
          {heading && <div className="dropdown__heading">{heading}</div>}
          <div className="dropdown__items">{children}</div>
        </div>
      </div>
    );
  }
}
