import { h, Component, createRef, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { Theme } from 'common/types';
import { sleep } from 'utils/sleep';
import { Button } from 'components/button';

type Props = RenderableProps<{
  title: string;
  titleClass?: string;
  heading?: string;
  isActive?: boolean;
  disabled?: boolean;
  buttonTitle?: string;
  onTitleClick?: () => void;
  mix?: string;
  theme: Theme;
  onOpen?: (root: HTMLDivElement) => unknown;
  onClose?: (root: HTMLDivElement) => unknown;
}>;

interface State {
  isActive: boolean;
  contentTranslateX: number;
}

export class Dropdown extends Component<Props, State> {
  rootNode = createRef<HTMLDivElement>();
  storedDocumentHeight: string | null = null;
  storedDocumentHeightSet = false;
  checkInterval: number | undefined = undefined;

  constructor(props: Props) {
    super(props);

    this.state = {
      isActive: props.isActive || false,
      contentTranslateX: 0,
    };

    this.onOutsideClick = this.onOutsideClick.bind(this);
    this.receiveMessage = this.receiveMessage.bind(this);
    this.__onOpen = this.__onOpen.bind(this);
    this.__onClose = this.__onClose.bind(this);
  }

  onTitleClick = () => {
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
          this.props.onOpen && this.props.onOpen(this.rootNode.current!);
        } else {
          this.__onClose();
          this.props.onClose && this.props.onClose(this.rootNode.current!);
        }

        if (this.props.onTitleClick) {
          this.props.onTitleClick();
        }
      }
    );
  };

  __onOpen() {
    const isChildOfDropDown = (() => {
      if (!this.rootNode.current) return false;
      let parent = this.rootNode.current.parentElement!;
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
      if (!this.rootNode.current || !this.state.isActive) return;
      const windowHeight = window.innerHeight;
      const dcBottom = (() => {
        // TODO: use ref
        const dc = Array.from(this.rootNode.current.children).find((c) => c.classList.contains('dropdown__content'));
        if (!dc) return 0;
        const rect = dc.getBoundingClientRect();
        return window.scrollY + Math.abs(rect.top) + dc.scrollHeight + 10;
      })();
      if (prevDcBottom === null && dcBottom <= windowHeight) return;
      if (dcBottom !== prevDcBottom) {
        prevDcBottom = dcBottom;
        document.body.style.minHeight = `${dcBottom}px`;
      }
    }, 100);
  }

  __onClose() {
    window.clearInterval(this.checkInterval);
    if (this.storedDocumentHeightSet) {
      document.body.style.minHeight = typeof this.storedDocumentHeight === 'string' ? this.storedDocumentHeight : '';
    }
  }

  async __adjustDropDownContent() {
    if (!this.rootNode.current) return;
    // TODO: use ref
    const dc = this.rootNode.current.querySelector<HTMLDivElement>('.dropdown__content');
    if (!dc) return;
    await sleep(0);
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
          this.props.onClose && this.props.onClose(this.rootNode.current!);
        }
      );
    } catch (e) {}
  }

  onOutsideClick(e: MouseEvent) {
    if (!this.rootNode.current || this.rootNode.current.contains(e.target as Node) || !this.state.isActive) return;
    this.setState(
      {
        contentTranslateX: 0,
        isActive: false,
      },
      () => {
        this.__onClose();
        this.props.onClose && this.props.onClose(this.rootNode.current!);
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

  render({ title, titleClass = '', heading, children, mix, theme, disabled, buttonTitle }: Props, { isActive }: State) {
    return (
      <div className={b('dropdown', { mix }, { theme, active: isActive })} ref={this.rootNode}>
        <Button
          aria-haspopup="listbox"
          aria-expanded={isActive && 'true'}
          onClick={this.onTitleClick}
          theme={theme}
          mix={['dropdown__title', titleClass]}
          kind="link"
          disabled={disabled}
          title={buttonTitle}
        >
          {title}
        </Button>
        {isActive && (
          <div
            className="dropdown__content"
            tabIndex={-1}
            role="listbox"
            style={{ transform: `translateX(${this.state.contentTranslateX}px)` }}
          >
            {heading && <div className="dropdown__heading">{heading}</div>}
            <div className="dropdown__items">{children}</div>
          </div>
        )}
      </div>
    );
  }
}
