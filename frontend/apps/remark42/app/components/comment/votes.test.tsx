import { createIntl } from 'common/intl';
import en from 'locales/en.json';

import { mountVotes } from './votes.js';

type Props = Parameters<typeof mountVotes>[1];
type Voted = { id: string; score: number };

const base: Props = { id: 'c1', vote: 0, votes: 3, controversy: undefined };

function deps(putVote: (id: string, vote: -1 | 1) => Promise<Voted> = () => new Promise<Voted>(() => {})) {
  return { intl: createIntl('en', en), putVote, patch: jest.fn(), positiveScore: () => false };
}

function mounted(putVote?: (id: string, vote: -1 | 1) => Promise<Voted>) {
  const root = document.createElement('span');
  const votes = mountVotes(root, base, deps(putVote));
  return { root, votes, up: () => (root.querySelector('[title="Vote up"]') as HTMLButtonElement).click() };
}

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Element[], before: Element[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

afterEach(() => {
  jest.restoreAllMocks();
  jest.useRealTimers();
});

describe('mountVotes', () => {
  it('keeps every node across a vote and an update', async () => {
    let answer = (_: Voted) => {};
    const { root, votes, up } = mounted(() => new Promise<Voted>((resolve) => (answer = resolve)));
    const before = Array.from(root.querySelectorAll('*'));

    up();
    answer({ id: 'c1', score: 4 });
    await Promise.resolve();
    votes.update({ ...base, vote: 1, votes: 4 });

    expectSameNodes(Array.from(root.querySelectorAll('*')), before);
  });

  it('shows no tooltip until a vote fails, then the error above the score until the pointer leaves', async () => {
    jest.useFakeTimers();
    let fail = (_: unknown) => {};
    const { root, up } = mounted(() => new Promise<never>((_, reject) => (fail = reject)));
    expect(root.querySelector('[role="tooltip"]')).toBeNull();

    up();
    fail({ code: 0 });
    await Promise.resolve();
    await Promise.resolve();

    const tooltip = root.querySelector('[role="tooltip"]')!;
    expect(tooltip.classList).toContain('top-left');
    expect(tooltip.textContent).toBe('Something went wrong. Please try again a bit later.');
    expect(tooltip.nextElementSibling).toBe(root.querySelector('[title="Score"]'));

    tooltip.dispatchEvent(new MouseEvent('mouseleave'));
    expect(root.querySelector('[role="tooltip"]')).toBeNull();
  });

  it('hides the error after ten seconds', async () => {
    jest.useFakeTimers();
    let fail = (_: unknown) => {};
    const { root, up } = mounted(() => new Promise<never>((_, reject) => (fail = reject)));
    up();
    fail({ code: 0 });
    await Promise.resolve();
    await Promise.resolve();

    jest.advanceTimersByTime(9_999);
    expect(root.querySelector('[role="tooltip"]')).not.toBeNull();
    jest.advanceTimersByTime(1);
    expect(root.querySelector('[role="tooltip"]')).toBeNull();
  });

  it('after destroy, leaves no timer and ignores a late answer', async () => {
    jest.useFakeTimers();
    let fail = (_: unknown) => {};
    const { root, votes, up } = mounted(() => new Promise<never>((_, reject) => (fail = reject)));
    up();
    const shown = root.innerHTML;

    votes.destroy();
    fail({ code: 0 });
    await Promise.resolve();
    await Promise.resolve();

    expect(jest.getTimerCount()).toBe(0);
    expect(root.innerHTML).toBe(shown);
  });

  it('after destroy, leaves no timer from a vote that had just succeeded', async () => {
    jest.useFakeTimers();
    const { votes, up } = mounted(() => Promise.resolve({ id: 'c1', score: 4 }));
    up();
    await Promise.resolve();
    await Promise.resolve();
    expect(jest.getTimerCount()).toBe(1);

    votes.destroy();

    expect(jest.getTimerCount()).toBe(0);
  });
});
