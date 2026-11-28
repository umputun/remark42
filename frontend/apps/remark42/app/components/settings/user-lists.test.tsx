import { h } from 'preact';
import { fireEvent } from '@testing-library/preact';

import { createIntl } from 'common/intl';
import type { BlockedUser, User } from 'common/types';
import en from 'locales/en.json';
import { render } from 'tests/utils';

import { Settings } from './settings';
import type { SettingsProps } from './user-lists.js';
import { mountSettings } from './user-lists.js';

const intl = createIntl('en', en);

function user(id: string, name: string, admin = false): User {
  return { id, name, admin, picture: '', ip: '', block: false, verified: false };
}

const hidden = { h1: user('h1', 'Hidden one'), h2: user('h2', 'Hidden two') };
const blocked: BlockedUser[] = [
  { id: 'b1', name: 'Blocked one', time: '2099-01-01T00:00:00Z' },
  { id: 'b2', name: 'Blocked two', time: '2099-01-01T00:00:00Z' },
];

function props(change: Partial<SettingsProps> = {}): SettingsProps {
  return {
    theme: 'light',
    intl,
    user: user('me', 'Me', true),
    blockedUsers: blocked,
    hiddenUsers: hidden,
    blockUser: jest.fn(() => Promise.resolve()),
    unblockUser: jest.fn(() => Promise.resolve()),
    hideUser: jest.fn(),
    unhideUser: jest.fn(),
    onUnblockSomeone: jest.fn(),
    ...change,
  };
}

function mount(change: Partial<SettingsProps> = {}) {
  const root = document.createElement('div');
  const settings = mountSettings(root, props(change));
  const action = (text: string, row = 0) =>
    Array.from(root.querySelectorAll<HTMLElement>('[role="button"]')).filter((el) => el.textContent === text)[row];
  const names = () =>
    Array.from(root.querySelectorAll('[title]'))
      .map((el) => el.textContent + (el.classList.contains('invisible') ? ' (dimmed)' : ''))
      .join(', ');
  return { root, settings, action, names };
}

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Element[], before: Element[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

beforeEach(() => {
  jest.spyOn(window, 'confirm').mockReturnValue(true);
});
afterEach(() => {
  jest.restoreAllMocks();
});

describe('mountSettings', () => {
  it('calls the handlers of the latest props', () => {
    const { settings, action } = mount();
    const latest = props();
    settings.update(latest);

    action('show').click();
    expect(latest.unhideUser).toHaveBeenCalledWith('h1');
    action('hide').click();
    expect(latest.hideUser).toHaveBeenCalledWith(hidden.h1);
    action('unblock', 1).click();
    expect(latest.unblockUser).toHaveBeenCalledWith('b2');
    action('block').click();
    expect(latest.blockUser).toHaveBeenCalledWith('b2', 'Blocked two', 'permanently');
    expect(latest.onUnblockSomeone).toHaveBeenCalledTimes(2);
  });

  it('asks before unblocking, and changes nothing when the answer is no', () => {
    const latest = props();
    const { settings, action, names } = mount();
    settings.update(latest);
    const before = names();
    jest.spyOn(window, 'confirm').mockReturnValue(false);

    action('unblock').click();

    expect(window.confirm).toHaveBeenCalledWith('Do you want to unblock Blocked one?');
    expect(latest.unblockUser).not.toHaveBeenCalled();
    expect(latest.onUnblockSomeone).not.toHaveBeenCalled();
    expect(names()).toBe(before);
  });

  it('asks before blocking again, and changes nothing when the answer is no', () => {
    const latest = props();
    const { settings, action, names } = mount();
    settings.update(latest);
    action('unblock').click();
    const before = names();
    jest.spyOn(window, 'confirm').mockReturnValue(false);

    action('block').click();

    expect(window.confirm).toHaveBeenLastCalledWith('Do you want to block Blocked one?');
    expect(latest.blockUser).not.toHaveBeenCalled();
    expect(names()).toBe(before);
    expect(action('block')).toBeDefined();
  });

  it('draws a row for each entry of a list that names one user twice', () => {
    const twice = { a: user('x', 'First'), b: user('x', 'Second') };

    const { names } = mount({ hiddenUsers: twice, blockedUsers: [blocked[0], blocked[0]] });

    expect(names()).toBe('First, Second, Blocked one, Blocked one');
  });

  it('keeps a selection inside the block period across an update', () => {
    const { root, settings } = mount();
    document.body.append(root);
    const period = root.querySelector('[aria-label="Blocked users"] li span:nth-child(2)')!.lastChild as Text;
    expect(period.data).toBe('permanently');
    window.getSelection()!.setBaseAndExtent(period, 2, period, 5);

    settings.update(props({ theme: 'dark' }));

    const selection = window.getSelection()!;
    expect([selection.anchorNode, selection.anchorOffset, selection.focusOffset]).toEqual([period, 2, 5]);
    root.remove();
  });

  it('draws a row each for ids that differ only in type', () => {
    const mixed = { a: { ...user('1', 'Text') }, b: { ...user('1', 'Number'), id: 1 as unknown as string } };

    const { names } = mount({ hiddenUsers: mixed });

    expect(names().startsWith('Text, Number')).toBe(true);
  });

  it('keeps every text node across an update that changes no text', () => {
    const { root, settings } = mount();
    const texts = () => {
      const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
      const found: Node[] = [];
      for (let node = walker.nextNode(); node; node = walker.nextNode()) {
        found.push(node);
      }
      return found;
    };
    const before = texts();

    settings.update(props({ theme: 'dark' }));

    const after = texts();
    expect(after).toHaveLength(before.length);
    after.forEach((node, i) => expect(node).toBe(before[i]));
  });

  it('dims a user shown again or unblocked, keeping them in the list with the way back', () => {
    const { action, names } = mount();

    action('show', 1).click();
    action('unblock').click();

    expect(names()).toBe('Hidden one, Hidden two (dimmed), Blocked one (dimmed), Blocked two');
    expect(action('hide')).toBeDefined();
    expect(action('block')).toBeDefined();
  });

  it.each([
    ['Enter', true],
    [' ', true],
    ['a', false],
  ])('works from the keyboard: %j', (key, acts) => {
    const props = mount();
    const button = props.action('show');
    const press = new KeyboardEvent('keypress', { key, cancelable: true });

    button.dispatchEvent(press);

    expect(button.textContent).toBe(acts ? 'hide' : 'show');
    expect(press.defaultPrevented).toBe(acts);
  });

  it('keeps every node across updates and actions, and the focus on the button pressed', () => {
    const { root, settings, action } = mount();
    document.body.append(root);
    const before = Array.from(root.querySelectorAll('*'));
    const unblock = action('unblock');
    unblock.focus();

    unblock.dispatchEvent(new KeyboardEvent('keypress', { key: 'Enter' }));
    action('show').click();
    settings.update(props({ theme: 'dark' }));

    expectSameNodes(Array.from(root.querySelectorAll('*')), before);
    expect(document.activeElement).toBe(unblock);
    expect(root.className).toBe('root themeDark');
    root.remove();
  });

  it('shows the lists it opened with, whatever later props say', () => {
    const { settings, names } = mount();
    const before = names();

    settings.update(props({ hiddenUsers: {}, blockedUsers: [blocked[1]] }));

    expect(names()).toBe(before);
  });

  it('shows the blocked users to an admin only, keeping the section for when they come back', () => {
    const { root, settings } = mount();
    const section = root.querySelector('[aria-label="Blocked users"]');

    settings.update(props({ user: user('me', 'Me') }));
    expect(root.querySelector('[aria-label="Blocked users"]')).toBeNull();

    settings.update(props());
    expect(root.querySelector('[aria-label="Blocked users"]')).toBe(section);
  });

  it('does nothing once destroyed', () => {
    const { settings, action } = mount();
    const latest = props();
    settings.update(latest);
    const show = action('show');

    settings.destroy();
    show.click();

    expect(latest.unhideUser).not.toHaveBeenCalled();
    expect(show.textContent).toBe('show');
  });
});

describe('<Settings /> in the preact tree', () => {
  it('keeps the plain nodes when the page renders again, and goes inert when it leaves', () => {
    const { theme: _, ...first } = props();
    const view = render(<Settings {...first} />);
    const before = Array.from(view.container.querySelectorAll('*'));
    const { theme: __, ...latest } = props();

    view.rerender(<Settings {...latest} />);
    expectSameNodes(Array.from(view.container.querySelectorAll('*')), before);
    fireEvent.click(view.getAllByText('show')[0]);
    expect(latest.unhideUser).toHaveBeenCalledWith('h1');

    const hide = view.getByText('hide');
    view.unmount();
    fireEvent.click(hide);
    expect(latest.hideUser).not.toHaveBeenCalled();
  });
});
