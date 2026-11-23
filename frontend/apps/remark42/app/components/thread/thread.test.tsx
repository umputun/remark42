import '@testing-library/jest-dom';

import { render } from 'tests/utils';
import { COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants';
import type { Comment } from 'common/types';
import type { StoreState } from 'store';

import { Thread } from './thread';

/** `[id, parent id]`, a parent before its replies and replies in the order the thread shows them */
type Reply = [id: string, pid: string];

function comment(id: string, pid: string, extra: Partial<Comment> = {}): Comment {
  return {
    id,
    pid,
    text: `<p>${id}</p>`,
    orig: id,
    score: 0,
    vote: 0,
    time: '2021-04-02T14:52:39.985281605-05:00',
    locator: { site: 'remark', url: 'https://example.com/post/' },
    user: { id: 'reader', name: 'Reader', picture: '', ip: '', admin: false, block: false, verified: false },
    ...extra,
  } as Comment;
}

function state(replies: Reply[], extra: Record<string, Partial<Comment>> = {}, collapsed: string[] = []) {
  const allComments: Record<string, Comment> = {};
  const childComments: Record<string, string[]> = {};

  for (const [id, pid] of replies) {
    allComments[id] = comment(id, pid, extra[id]);
    if (pid) {
      (childComments[pid] ||= []).push(id);
    }
  }

  return {
    comments: {
      sort: '-active',
      isFetching: false,
      topComments: [replies[0][0]],
      pinnedComments: [],
      activeComment: null,
      allComments,
      childComments,
    },
    collapsedThreads: Object.fromEntries(collapsed.map((id) => [id, true])),
    theme: 'light',
    info: { url: 'https://example.com/post/', count: replies.length, read_only: false },
    hiddenUsers: {},
    bannedUsers: [],
    user: null,
  } as unknown as Partial<StoreState>;
}

/** a comment and `length` replies, each answering the one before it */
function chain(length: number, from = ''): Reply[] {
  const replies: Reply[] = [];
  let pid = from;

  for (let i = 0; i <= length; i++) {
    const id = `chain-${i}`;
    replies.push([id, pid]);
    pid = id;
  }

  return replies;
}

function renderThread(...args: Parameters<typeof state>) {
  const [replies] = args;

  return render(
    <div role="list">
      <Thread id={replies[0][0]} level={0} getPreview={async () => ''} />
    </div>,
    state(...args)
  );
}

/** ids of the comments in the order the page shows them */
function shownIds(container: Element): string[] {
  return Array.from(container.querySelectorAll('article[id]'), (article) =>
    article.id.slice(COMMENT_NODE_CLASSNAME_PREFIX.length)
  );
}

/** how many threads the thread of the comment stands inside */
function depth(container: Element, id: string): number {
  let depth = 0;
  let item = container.querySelector(`#${COMMENT_NODE_CLASSNAME_PREFIX}${id}`)?.closest('[role="listitem"]');

  while ((item = item?.parentElement?.closest('[role="listitem"]'))) {
    depth++;
  }

  return depth;
}

/** the list a thread keeps the replies to its comment in */
function repliesOf(container: Element, id: string): Element | null | undefined {
  return container
    .querySelector(`#${COMMENT_NODE_CLASSNAME_PREFIX}${id}`)
    ?.closest('[role="listitem"]')
    ?.querySelector(':scope > [role="list"]');
}

describe('<Thread />', () => {
  const { IntersectionObserver } = window;

  beforeEach(() => {
    // no comment comes into view, so each one is the placeholder a comment off screen is
    window.IntersectionObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof window.IntersectionObserver;
  });

  afterEach(() => {
    window.IntersectionObserver = IntersectionObserver;
  });

  it('draws every reply of a chain far longer than the stack can nest', () => {
    const replies = chain(1500);
    const { container } = renderThread(replies);

    expect(shownIds(container)).toEqual(replies.map(([id]) => id));
  });

  it('nests replies down to the deepest indent and no further', () => {
    const { container } = renderThread(chain(9));

    expect([0, 1, 2, 3, 4, 5, 6, 7, 8, 9].map((i) => depth(container, `chain-${i}`))).toEqual([
      0, 1, 2, 3, 4, 5, 6, 6, 6, 6,
    ]);
    // the replies below the deepest indent are items of the list of the thread just above it
    expect(repliesOf(container, 'chain-5')?.children).toHaveLength(4);
    expect(repliesOf(container, 'chain-6')).toBeNull();
  });

  describe('below the deepest indent', () => {
    // chain-5 is the thread just above the deepest indent
    const deep: Reply[] = [
      ...chain(5),
      ['a', 'chain-5'],
      ['b', 'a'],
      ['c', 'b'],
      ['d', 'b'],
      ['e', 'a'],
      ['f', 'e'],
      ['g', 'chain-5'],
    ];
    const above = chain(5).map(([id]) => id);

    it('keeps the replies in the order of the thread', () => {
      const { container } = renderThread(deep);

      expect(shownIds(container)).toEqual([...above, 'a', 'b', 'c', 'd', 'e', 'f', 'g']);
      expect(repliesOf(container, 'chain-5')?.children).toHaveLength(7);
    });

    it('leaves out the replies to a collapsed comment', () => {
      const { container } = renderThread(deep, {}, ['b']);

      expect(shownIds(container)).toEqual([...above, 'a', 'b', 'e', 'f', 'g']);
    });

    it('leaves out a comment of a hidden user with the replies to it', () => {
      const { container } = renderThread(deep, { e: { hidden: true } });

      expect(shownIds(container)).toEqual([...above, 'a', 'b', 'c', 'd', 'g']);
    });

    it('draws no list when every reply is hidden', () => {
      const { container } = renderThread(deep, { a: { hidden: true }, g: { hidden: true } });

      expect(shownIds(container)).toEqual(above);
      expect(repliesOf(container, 'chain-5')).toBeNull();
    });
  });
});
