import { h } from 'preact';
import { act } from '@testing-library/preact';

import { render } from 'tests/utils';

import type { Props } from './comment-actions';
import { CommentActions } from './comment-actions';

const NOW = new Date('2026-11-27T12:00:00Z').getTime();

function props(change: Partial<Props> = {}): Props {
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

beforeEach(() => {
  jest.useFakeTimers({ now: NOW });
});
afterEach(() => {
  jest.useRealTimers();
});

// the markup the preact component drew, recorded before it was replaced; innerHTML keeps the
// attributes in the order they were written
describe('the comment actions draw the markup of the preact component they replace', () => {
  it.each<[string, Partial<Props>]>([
    ["someone else's comment", {}],
    ['replying', { replying: true }],
    ['read-only', { readOnly: true }],
    ['own comment, editable for five minutes', { currentUser: true, editable: true, editDeadline: NOW + 300_000 }],
    ['own comment, being edited', { currentUser: true, editable: true, editing: true, editDeadline: NOW + 300_000 }],
    ['own comment, editable without a deadline', { currentUser: true, editable: true, editDeadline: Infinity }],
    ['own comment, no longer editable', { currentUser: true, editable: false, editDeadline: NOW + 300_000 }],
    ['admin', { admin: true }],
    ['admin, copied and pinned', { admin: true, copied: true, pinned: true }],
    ['admin, the author blocked', { admin: true, bannedUser: true }],
    ['admin, own comment', { admin: true, currentUser: true, editable: true, editDeadline: NOW + 300_000 }],
  ])('%s', (_, change) => {
    expect(render(<CommentActions {...props(change)} />).container.innerHTML).toMatchSnapshot();
  });

  it('as the edit countdown runs out', () => {
    const onDisableEditing = jest.fn();
    const { container } = render(
      <CommentActions {...props({ currentUser: true, editable: true, editDeadline: NOW + 3_000, onDisableEditing })} />
    );

    act(() => {
      jest.advanceTimersByTime(1_000);
    });
    expect(container.innerHTML).toMatchSnapshot('a second later');
    act(() => {
      jest.advanceTimersByTime(2_000);
    });
    expect(container.innerHTML).toMatchSnapshot('at the deadline');
    expect(onDisableEditing).toHaveBeenCalledTimes(1);
  });
});
