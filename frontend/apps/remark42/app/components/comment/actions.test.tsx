import { h } from 'preact';
import { fireEvent } from '@testing-library/preact';

import { createIntl } from 'common/intl';
import en from 'locales/en.json';
import { render } from 'tests/utils';

import { CommentActions } from './comment-actions';
import type { ActionsProps } from './actions.js';
import { mountActions } from './actions.js';

const NOW = new Date('2026-11-27T12:00:00Z').getTime();
const intl = createIntl('en', en);

function props(change: Partial<ActionsProps> = {}): ActionsProps {
  return {
    admin: false,
    currentUser: false,
    pinned: false,
    copied: false,
    bannedUser: false,
    readOnly: false,
    editing: false,
    replying: false,
    editable: false,
    editDeadline: undefined,
    onCopy: jest.fn(),
    onToggleEditing: jest.fn(),
    onDelete: jest.fn(),
    onTogglePin: jest.fn(),
    onToggleReplying: jest.fn(),
    onHideUser: jest.fn(),
    onBlockUser: jest.fn(),
    onUnblockUser: jest.fn(),
    onDisableEditing: jest.fn(),
    ...change,
  };
}

function mount(change: Partial<ActionsProps> = {}) {
  const root = document.createElement('div');
  const actions = mountActions(root, props(change), intl);
  const button = (text: string) =>
    Array.from(root.querySelectorAll('button')).find((b) => b.textContent === text) as HTMLButtonElement;
  return { root, actions, button };
}

const editable = { currentUser: true, editable: true };

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Element[], before: Element[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

beforeEach(() => {
  jest.useFakeTimers({ now: NOW });
});
afterEach(() => {
  jest.useRealTimers();
});

describe('mountActions', () => {
  it('calls the handlers of the latest props', () => {
    const { actions, button } = mount({ admin: true });
    const latest = props({ admin: true });
    actions.update(latest);

    for (const [text, handler] of [
      ['Reply', latest.onToggleReplying],
      ['Hide user', latest.onHideUser],
      ['Copy', latest.onCopy],
      ['Pin', latest.onTogglePin],
      ['Delete', latest.onDelete],
    ] as const) {
      button(text).click();
      expect(handler).toHaveBeenCalledTimes(1);
    }
    actions.update({ ...latest, editable: true, editDeadline: Infinity, bannedUser: true });
    button('Edit').click();
    button('Unblock').click();
    expect(latest.onToggleEditing).toHaveBeenCalledTimes(1);
    expect(latest.onUnblockUser).toHaveBeenCalledTimes(1);
  });

  it('blocks the author for the period picked, keeping the label', () => {
    const onBlockUser = jest.fn();
    const { root } = mount({ admin: true, onBlockUser });
    const select = root.querySelector('select')!;

    select.value = '1440m';
    select.dispatchEvent(new Event('change'));

    expect(onBlockUser).toHaveBeenCalledWith('1440m');
    expect(root.querySelector('[data-testid="select-root"]')!.firstChild!.textContent).toBe('Block');
    expect(select.value).toBe('1440m');
  });

  it.each<[string, Partial<ActionsProps>]>([
    ['own comment, editable', { ...editable, editDeadline: NOW + 60_000 }],
    ['an admin, with the block select', { admin: true }],
    ["an admin's own comment, editable", { ...editable, admin: true, editDeadline: NOW + 60_000 }],
  ])('keeps every node across updates: %s', (_, change) => {
    const { root, actions } = mount(change);
    const before = Array.from(root.querySelectorAll('*'));

    actions.update(props(change));
    jest.advanceTimersByTime(1_000);

    expectSameNodes(Array.from(root.querySelectorAll('*')), before);
  });

  it('offers the periods afresh after the author is unblocked', () => {
    const onBlockUser = jest.fn();
    const { root, actions } = mount({ admin: true, onBlockUser });
    const picked = root.querySelector('select')!;
    picked.value = '1440m';
    picked.dispatchEvent(new Event('change'));

    actions.update(props({ admin: true, onBlockUser, bannedUser: true }));
    actions.update(props({ admin: true, onBlockUser, bannedUser: false }));

    // a select still showing the earlier pick would fire no change when it is picked again
    expect(root.querySelector('select')!.selectedIndex).toBe(0);
  });

  it('keeps the focus on a button that stays when the edit controls go', () => {
    const { root, actions, button } = mount({ ...editable, editDeadline: NOW + 60_000 });
    document.body.append(root);
    button('Reply').focus();

    actions.update(props({ ...editable, editable: false, editDeadline: NOW + 60_000 }));

    expect(button('Edit')).toBeUndefined();
    expect(document.activeElement).toBe(button('Reply'));
    root.remove();
  });
});

describe('the edit countdown', () => {
  it('calls onDisableEditing once a deadline that has already passed is shown', () => {
    const onDisableEditing = jest.fn();
    const { root } = mount({ ...editable, editDeadline: NOW - 1, onDisableEditing });

    expect(root.querySelector('[role="timer"]')!.textContent).toBe('0s');
    expect(onDisableEditing).toHaveBeenCalledTimes(1);
    expect(jest.getTimerCount()).toBe(0);
  });

  it('starts again from a new deadline', () => {
    const onDisableEditing = jest.fn();
    const { root, actions } = mount({ ...editable, editDeadline: NOW + 3_000, onDisableEditing });
    jest.advanceTimersByTime(2_000);

    actions.update(props({ ...editable, editDeadline: NOW + 10_000, onDisableEditing }));
    expect(root.querySelector('[role="timer"]')!.textContent).toBe('8s');
    jest.advanceTimersByTime(7_000);
    expect(onDisableEditing).not.toHaveBeenCalled();
    jest.advanceTimersByTime(1_000);
    expect(onDisableEditing).toHaveBeenCalledTimes(1);
  });

  it('runs no timer for a deadline that never comes', () => {
    const { root } = mount({ ...editable, editDeadline: Infinity });

    expect(root.querySelector('[role="timer"]')).toBeNull();
    expect(jest.getTimerCount()).toBe(0);
  });

  it('stops when editing is no longer offered, and on destroy', () => {
    const { actions } = mount({ ...editable, editDeadline: NOW + 60_000 });
    expect(jest.getTimerCount()).toBe(1);

    actions.update(props({ ...editable, editable: false, editDeadline: NOW + 60_000 }));
    expect(jest.getTimerCount()).toBe(0);

    actions.update(props({ ...editable, editDeadline: NOW + 60_000 }));
    expect(jest.getTimerCount()).toBe(1);
    actions.destroy();
    expect(jest.getTimerCount()).toBe(0);
  });
});

describe('<CommentActions /> in the preact tree', () => {
  it('keeps the plain nodes when the comment renders again, and stops the countdown when it leaves', () => {
    const view = render(<CommentActions {...props({ ...editable, editDeadline: NOW + 60_000 })} />);
    const before = Array.from(view.container.querySelectorAll('*'));
    const onDelete = jest.fn();

    view.rerender(<CommentActions {...props({ ...editable, editDeadline: NOW + 60_000, onDelete })} />);
    expectSameNodes(Array.from(view.container.querySelectorAll('*')), before);
    fireEvent.click(view.getByText('Delete'));
    expect(onDelete).toHaveBeenCalledTimes(1);

    view.unmount();
    expect(jest.getTimerCount()).toBe(0);
  });
});
