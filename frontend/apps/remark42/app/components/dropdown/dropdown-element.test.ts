import type { DropdownControl, DropdownProps } from './dropdown-element.js';
import { dropdownElement } from './dropdown-element.js';

const base: DropdownProps = { title: 'Email', buttonTitle: 'Subscribe by Email', theme: 'light' };

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Element[], before: Element[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

function mount(props: Partial<DropdownProps> = {}, inner = 'Inner') {
  const root = document.createElement('div');
  document.body.append(root);
  let control: DropdownControl = { close: () => {}, keepOpen: () => {} };
  const destroyContent = jest.fn();
  const opened = jest.fn();
  const dropdown = dropdownElement(root, { ...base, ...props }, (items, given) => {
    control = given;
    opened();
    const button = document.createElement('button');
    button.type = 'button';
    button.textContent = inner;
    items.replaceChildren(button);
    return { destroy: destroyContent };
  });
  const title = root.querySelector('button') as HTMLButtonElement;
  return {
    root,
    dropdown,
    title,
    opened,
    destroyContent,
    control: () => control,
    panel: () => root.querySelector('div[role="listbox"]'),
    inner: () => root.querySelector('div[role="listbox"] button'),
  };
}

const tick = () => new Promise((resolve) => setTimeout(resolve, 0));

afterEach(() => {
  document.body.replaceChildren();
  document.body.style.minHeight = '';
  jest.useRealTimers();
  jest.restoreAllMocks();
});

describe('dropdownElement', () => {
  it('opens with its title button and closes on an outside click', () => {
    const { title, panel, root } = mount();

    title.click();
    expect(panel()).not.toBeNull();
    expect(title.getAttribute('aria-expanded')).toBe('true');
    expect(root.className).toContain('active');

    document.body.click();
    expect(panel()).toBeNull();
    expect(title.getAttribute('aria-expanded')).toBe('false');
  });

  it('stays open when an inner click detaches the clicked node', () => {
    const { title, panel, inner } = mount();
    title.click();
    const clicked = inner() as HTMLElement;
    // what a step change inside the content does: the clicked node leaves before the click
    // finishes its way to the document
    panel()!.addEventListener('click', (evt) => (evt.target as HTMLElement).remove(), { capture: true });

    clicked.click();

    expect(panel()).not.toBeNull();
  });

  it('stays open on an outside click and on the clickOutside message while the content keeps it open', async () => {
    const { title, panel, control } = mount();
    title.click();
    control().keepOpen(true);

    document.body.click();
    window.dispatchEvent(new MessageEvent('message', { data: { clickOutside: true }, source: window.parent }));
    await tick();
    expect(panel()).not.toBeNull();

    control().keepOpen(false);
    window.dispatchEvent(new MessageEvent('message', { data: { clickOutside: true }, source: window.parent }));
    expect(panel()).toBeNull();
  });

  it('is closed by its content', () => {
    const { title, panel, control, destroyContent } = mount();
    title.click();

    control().close();

    expect(panel()).toBeNull();
    expect(destroyContent).toHaveBeenCalledTimes(1);
  });

  it('builds the content afresh on each opening and destroys it on each closing', () => {
    const { title, opened, destroyContent } = mount();

    title.click();
    title.click();
    title.click();

    expect(opened).toHaveBeenCalledTimes(2);
    expect(destroyContent).toHaveBeenCalledTimes(1);
  });

  it('closes a dropdown kept open when the reader opens another, and puts the page height back', async () => {
    const first = mount({ buttonTitle: 'First' }, 'Waiting');
    const second = mount({ buttonTitle: 'Second' }, 'Other');
    document.body.style.minHeight = '';

    first.title.click();
    // opening finishes a tick later, and only then saves the page height
    await tick();
    first.control().keepOpen(true);
    // what the first dropdown's own check does on a short page, which jsdom has no layout for
    document.body.style.minHeight = '860px';

    document.body.click();
    expect(first.panel()).not.toBeNull();

    second.title.click();
    await tick();
    expect(first.panel()).toBeNull();
    expect(second.panel()).not.toBeNull();
    expect(document.body.style.minHeight).toBe('');

    document.body.click();
    expect(second.panel()).toBeNull();
    expect(document.body.style.minHeight).toBe('');
  });

  it('leaves a dropdown inside another open when it opens', () => {
    const outer = mount({ buttonTitle: 'Outer' });
    outer.title.click();
    const nestedRoot = document.createElement('div');
    outer.panel()!.append(nestedRoot);
    const nested = dropdownElement(nestedRoot, { ...base, buttonTitle: 'Nested' }, () => ({ destroy() {} }));

    (nestedRoot.querySelector('button') as HTMLButtonElement).click();

    expect(outer.panel()).not.toBeNull();
    nested.destroy();
  });

  it('puts the page height back, and stops, when a dropdown kept open is destroyed', async () => {
    jest.useFakeTimers();
    const { title, dropdown, control, destroyContent } = mount();
    document.body.style.minHeight = '';
    title.click();
    jest.advanceTimersByTime(0);
    await Promise.resolve();
    control().keepOpen(true);
    document.body.style.minHeight = '860px';
    expect(jest.getTimerCount()).toBe(1);

    // signing out replaces the subscription buttons with the sign-in form while the panel is open
    dropdown.destroy();

    expect(document.body.style.minHeight).toBe('');
    expect(jest.getTimerCount()).toBe(0);
    expect(destroyContent).toHaveBeenCalledTimes(1);
  });

  it('removes its document and window listeners when destroyed', () => {
    const documentAdd = jest.spyOn(document, 'addEventListener');
    const documentRemove = jest.spyOn(document, 'removeEventListener');
    const windowAdd = jest.spyOn(window, 'addEventListener');
    const windowRemove = jest.spyOn(window, 'removeEventListener');
    const { dropdown } = mount();
    expect(documentAdd).toHaveBeenCalled();
    expect(windowAdd).toHaveBeenCalled();

    dropdown.destroy();

    // the capture flag is part of what identifies a listener: removing one without it leaves it in place
    const listeners = (spy: jest.SpyInstance) =>
      spy.mock.calls.map(([type, listener, options]) => [
        type,
        listener,
        typeof options === 'boolean' ? options : Boolean(options?.capture),
      ]);
    expect(listeners(documentRemove)).toEqual(listeners(documentAdd));
    expect(listeners(windowRemove)).toEqual(listeners(windowAdd));
  });

  it('keeps its nodes across updates and applies the latest props', () => {
    const { root, title, dropdown } = mount();
    title.click();
    const before = Array.from(root.querySelectorAll('*'));

    dropdown.update({
      ...base,
      title: 'Mail',
      buttonTitle: 'Only for you',
      theme: 'dark',
      disabled: true,
      mix: 'extra',
    });

    expectSameNodes(Array.from(root.querySelectorAll('*')), before);
    expect(title.textContent).toBe('Mail');
    expect(title.title).toBe('Only for you');
    expect(title.disabled).toBe(true);
    expect(title.className).toContain('themeDark');
    expect(root.className).toBe('root active themeDark extra');
  });
});
