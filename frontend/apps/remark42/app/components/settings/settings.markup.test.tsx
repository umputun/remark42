import { h } from 'preact';
import { fireEvent } from '@testing-library/preact';

import { createIntl } from 'common/intl';
import type { BlockedUser, User } from 'common/types';
import en from 'locales/en.json';
import { render } from 'tests/utils';

import { Settings } from './settings';

type Props = Parameters<typeof Settings>[0];

const NOW = new Date('2026-11-28T12:00:00Z').getTime();
const intl = createIntl('en', en);
const until = new Date('2026-12-05T09:30:00Z');

function user(id: string, name: string, admin = false): User {
  return { id, name, admin, picture: '', ip: '', block: false, verified: false };
}

const hidden = { h1: user('h1', 'Hidden one'), h2: user('h2', '') };
const blocked: BlockedUser[] = [
  { id: 'b1', name: 'Blocked for a week', time: until.toISOString() },
  { id: 'b2', name: '', time: '2099-01-01T00:00:00Z' },
];

function props(change: Partial<Props> = {}): Props {
  return {
    intl,
    user: user('me', 'Me'),
    blockedUsers: [],
    hiddenUsers: {},
    blockUser: jest.fn(() => Promise.resolve()),
    unblockUser: jest.fn(() => Promise.resolve()),
    hideUser: jest.fn(),
    unhideUser: jest.fn(),
    onUnblockSomeone: jest.fn(),
    ...change,
  };
}

// the block period reads in the machine's time zone; the snapshot holds a placeholder for it, so it
// records the markup and not the zone the run happened in
function html(container: Element) {
  const period = `until ${intl.formatDate(until)} at ${intl.formatTime(until)}`;
  return container.innerHTML.split(period).join('until {day} at {time}');
}

function action(container: Element, text: string, row: number) {
  return Array.from(container.querySelectorAll('[role="button"]')).filter((el) => el.textContent === text)[row];
}

beforeEach(() => {
  jest.useFakeTimers({ now: NOW });
  jest.spyOn(window, 'confirm').mockReturnValue(true);
});
afterEach(() => {
  jest.useRealTimers();
  jest.restoreAllMocks();
});

// the markup the preact component drew, recorded before it was replaced; innerHTML keeps the
// attributes in the order they were written
describe('the settings draw the markup of the preact component they replace', () => {
  it.each<[string, Partial<Props>, object]>([
    ['a reader with no hidden users', {}, {}],
    ['a reader with hidden users', { hiddenUsers: hidden }, {}],
    ['an admin with nobody blocked', { user: user('me', 'Me', true) }, {}],
    [
      'an admin with hidden and blocked users',
      { user: user('me', 'Me', true), hiddenUsers: hidden, blockedUsers: blocked },
      {},
    ],
    ['the dark theme', { hiddenUsers: hidden }, { theme: 'dark' }],
    ['the light theme', { hiddenUsers: hidden }, { theme: 'light' }],
    ['signed out', { user: null, hiddenUsers: hidden, blockedUsers: blocked }, {}],
  ])('%s', (_, change, store) => {
    expect(html(render(<Settings {...props(change)} />, store).container)).toMatchSnapshot();
  });

  it('after showing a hidden user, and hiding them again', () => {
    const { container } = render(<Settings {...props({ hiddenUsers: hidden })} />);

    fireEvent.click(action(container, 'show', 0));
    expect(html(container)).toMatchSnapshot('shown');
    fireEvent.click(action(container, 'hide', 0));
    expect(html(container)).toMatchSnapshot('hidden again');
  });

  it('after unblocking a user, and blocking them again', () => {
    const admin = props({ user: user('me', 'Me', true), blockedUsers: blocked });
    const { container } = render(<Settings {...admin} />);

    fireEvent.click(action(container, 'unblock', 1));
    expect(html(container)).toMatchSnapshot('unblocked');
    fireEvent.click(action(container, 'block', 0));
    expect(html(container)).toMatchSnapshot('blocked again');
  });

  it('when the admin does not confirm an unblock', () => {
    jest.spyOn(window, 'confirm').mockReturnValue(false);
    const { container } = render(<Settings {...props({ user: user('me', 'Me', true), blockedUsers: blocked })} />);

    fireEvent.click(action(container, 'unblock', 0));
    expect(html(container)).toMatchSnapshot();
  });
});
