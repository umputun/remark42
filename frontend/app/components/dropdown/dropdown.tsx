/** @jsx h */
import { Component, h, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { Button } from '@app/components/button';
import { Theme } from '@app/common/types';
import { sleep } from '@app/utils/sleep';
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
  selectableItems?: string[];
  activeSelectableItemID?: number;
  onDropdownItemClick?: (e: Event) => void;
}

interface State {
  isActive: boolean;
  contentTranslateX: number;
  activeSelectableItemID: number;
  selectableItems?: string[];
  selectableItemsFilter?: string;
}

export default class Dropdown extends Component<Props, State> {
  rootNode?: HTMLDivElement;

  constructor(props: Props) {
    super(props);

    const { isActive, selectableItems } = this.props;

    let { activeSelectableItemID } = this.props;
    if (activeSelectableItemID === undefined) {
      activeSelectableItemID = 0;
    }

    this.state = {
      isActive: isActive || false,
      contentTranslateX: 0,
      activeSelectableItemID,
      selectableItems,
    };

    this.onOutsideClick = this.onOutsideClick.bind(this);
    this.receiveMessage = this.receiveMessage.bind(this);
    this.__onOpen = this.__onOpen.bind(this);
    this.__onClose = this.__onClose.bind(this);
    this.forceOpen = this.forceOpen.bind(this);
    this.forceClose = this.forceClose.bind(this);
    this.selectNextSelectableItem = this.selectNextSelectableItem.bind(this);
    this.selectPreviousSelectableItem = this.selectPreviousSelectableItem.bind(this);
    this.setSelectableItemsFilter = this.setSelectableItemsFilter.bind(this);
    this.onDropdownItemHover = this.onDropdownItemHover.bind(this);
  }

  selectNextSelectableItem() {
    const { selectableItems, activeSelectableItemID } = this.state;

    if (!selectableItems) return;

    const itemsLength = selectableItems.length;
    const firstItem = 0;

    let newActiveSelectableItemID = activeSelectableItemID + 1;

    if (newActiveSelectableItemID >= itemsLength) {
      newActiveSelectableItemID = firstItem;
    }

    this.setState({
      activeSelectableItemID: newActiveSelectableItemID,
    });
  }

  selectPreviousSelectableItem() {
    const { selectableItems, activeSelectableItemID } = this.state;

    if (!selectableItems) return;

    const itemsLength = selectableItems.length;
    const lastItem = itemsLength - 1;

    let newActiveSelectableItemID = activeSelectableItemID - 1;

    if (newActiveSelectableItemID < 0) {
      newActiveSelectableItemID = lastItem;
    }

    this.setState({
      activeSelectableItemID: newActiveSelectableItemID,
    });
  }

  setSelectableItemsFilter(selectableItemsFilter?: string) {
    this.setState({
      selectableItemsFilter,
    });
  }

  getSelectedItem() {
    const { selectableItems, activeSelectableItemID } = this.state;

    if (!selectableItems || activeSelectableItemID === undefined) return;

    return selectableItems[activeSelectableItemID];
  }

  filterSelectableItems(): void {
    const { selectableItemsFilter } = this.state;
    if (!selectableItemsFilter) return;

    const selectableItems = this.props.selectableItems || [];

    const filteredSelectableItems = selectableItems.filter(selectableItem => {
      return ~selectableItem.indexOf(selectableItemsFilter);
    });

    this.setState({
      selectableItems: filteredSelectableItems,
    });
  }

  generateSelectableItems(selectableItems: string[]) {
    if (!this.props.onDropdownItemClick) return;

    if (selectableItems.length === 0) {
      return <DropdownItem>No such item</DropdownItem>;
    }

    return selectableItems.map((selectableItem, index) => {
      return (
        <DropdownItem
          index={index}
          onFocus={this.onDropdownItemHover}
          onMouseOver={this.onDropdownItemHover}
          active={index === this.state.activeSelectableItemID}
          selectable={true}
          onClick={this.props.onDropdownItemClick}
        >
          {selectableItem}
        </DropdownItem>
      );
    });
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

  forceOpen() {
    if (this.state.isActive) return;

    this.setState({
      isActive: true,
    });
    this.__adjustDropDownContent().then(() => this.__onOpen());
  }

  __onClose() {
    const { selectableItems } = this.props;

    window.clearInterval(this.checkInterval);
    if (this.storedDocumentHeightSet) {
      document.body.style.minHeight = this.storedDocumentHeight;
    }

    this.setState({
      activeSelectableItemID: 0,
      selectableItemsFilter: undefined,
      selectableItems,
    });
  }

  forceClose() {
    if (!this.state.isActive) return;

    this.setState({
      isActive: false,
    });
    this.__onClose();
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

  onDropdownItemHover(e: Event) {
    const target = e.target;

    if (target instanceof HTMLElement) {
      const { id } = target.dataset;

      if (id !== undefined) {
        this.setState({
          activeSelectableItemID: +id,
        });
      }
    }
  }

  render(props: RenderableProps<Props>, { isActive }: State) {
    let { children } = props;
    const { title, titleClass, heading, mix } = props;

    {
      if (this.state.selectableItems) {
        children = this.generateSelectableItems(this.state.selectableItems);
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
