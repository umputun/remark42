import '@testing-library/jest-dom';
import { waitFor } from '@testing-library/preact';

import { render } from 'tests/utils';
import * as api from 'common/api';
import * as postMessage from 'utils/post-message';
import type { Comment, User } from 'common/types';
import type { StoreState } from 'store';

import { ConnectedRoot } from './root';

let mockMobile = false;
// the "Show more" button exists only on a phone, and Google crawls with a phone's user agent
jest.mock('common/constants', () => ({
  ...jest.requireActual('common/constants'),
  get IS_MOBILE() {
    return mockMobile;
  },
}));
let mockMaxShown: number | undefined;
jest.mock('common/settings', () => {
  const actual = jest.requireActual('common/settings');
  return {
    ...actual,
    get maxShownComments() {
      return mockMaxShown ?? actual.maxShownComments;
    },
  };
});

const stateStub: Partial<StoreState> = {
  comments: {
    sort: '-active',
    isFetching: false,
    childComments: {},
    topComments: [],
    pinnedComments: [],
    allComments: {},
    activeComment: null,
  },
  collapsedThreads: {},
  theme: 'light',
  info: { url: 'test-url', count: 0, read_only: false },
  hiddenUsers: {},
  bannedUsers: [],
  user: null,
};

describe('<ConnectedRoot />', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('reports iframe height only after the initial user fetch settles', async () => {
    let resolveUser!: (user: User | null) => void;
    jest.spyOn(api, 'getUser').mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveUser = resolve;
        })
    );
    // keep comments loading so only the user fetch controls the first height report
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
    const updateIframeHeight = jest.spyOn(postMessage, 'updateIframeHeight').mockImplementation(() => undefined);

    render(<ConnectedRoot />, stateStub);

    // while the global preloader is shown, no height must be sent to the parent page,
    // otherwise the parent shrinks the iframe to the preloader size and it blinks
    expect(updateIframeHeight).not.toHaveBeenCalled();

    resolveUser(null);

    await waitFor(() => expect(updateIframeHeight).toHaveBeenCalled());
  });

  it('falls back to reporting iframe height when the user fetch hangs', () => {
    jest.useFakeTimers();
    try {
      jest.spyOn(api, 'getUser').mockImplementation(() => new Promise(() => undefined));
      jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
      const updateIframeHeight = jest.spyOn(postMessage, 'updateIframeHeight').mockImplementation(() => undefined);

      render(<ConnectedRoot />, stateStub);
      expect(updateIframeHeight).not.toHaveBeenCalled();

      jest.advanceTimersByTime(5000);
      expect(updateIframeHeight).toHaveBeenCalled();
    } finally {
      jest.useRealTimers();
    }
  });

  it('sets dir on the root element from the active locale (direction mapping itself is covered by common/direction.test.ts)', () => {
    jest.spyOn(api, 'getUser').mockImplementation(() => new Promise(() => undefined));
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));

    const { container } = render(<ConnectedRoot />, stateStub, 'he');

    expect(container.firstElementChild).toHaveAttribute('dir', 'rtl');
  });

  const reader: User = {
    id: 'reader',
    name: 'reader name',
    picture: '',
    ip: '',
    admin: false,
    block: false,
    verified: false,
  };

  /** renders a thread with a reply, a deleted comment and `more` further comments, all in view */
  async function renderThreads(user: User | null, { more = 0, hideReply = false } = {}) {
    jest.spyOn(api, 'getUser').mockResolvedValue(user);
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
    // every comment is in view, as on the tall page a crawler renders; off screen one is a placeholder
    window.IntersectionObserver = class {
      constructor(private callback: IntersectionObserverCallback) {}
      observe(target: Element) {
        // InView registers the element only after it starts observing it
        setTimeout(() => this.callback([{ target, isIntersecting: true } as IntersectionObserverEntry], this as never));
      }
      unobserve() {}
      disconnect() {}
    } as unknown as typeof IntersectionObserver;

    const comment = (id: string, pid: string, text: string, extra: Partial<Comment> = {}): Comment => ({
      id,
      pid,
      text,
      orig: text,
      score: 1,
      vote: 0,
      controversy: 0,
      voted_ips: [],
      time: '2026-11-21T18:05:00Z',
      locator: { url: 'http://localhost/post/1', site: 'remark' },
      user: { id: `user_${id}`, name: `name ${id}`, picture: '', ip: '', admin: false, block: false, verified: false },
      ...extra,
    });
    const extra = Array.from({ length: more }, (_, i) => comment(`m${i}`, '', `<p>more ${i}</p>`));
    const comments = {
      c1: comment('c1', '', '<p>first comment with <a href="https://example.com/">a link</a></p>'),
      c2: comment('c2', 'c1', '<p>a reply</p>', { hidden: hideReply }),
      c3: comment('c3', '', '', { delete: true }),
      ...Object.fromEntries(extra.map((c) => [c.id, c])),
    };

    const { container } = render(<ConnectedRoot />, {
      ...stateStub,
      user,
      comments: {
        ...stateStub.comments!,
        topComments: ['c1', 'c3', ...extra.map((c) => c.id)],
        childComments: { c1: ['c2'] },
        allComments: comments,
      },
    });

    // a comment out of view renders as an empty placeholder article
    await waitFor(() => {
      const articles = Array.from(container.querySelectorAll('article'));
      expect(articles.length).toBeGreaterThanOrEqual(hideReply ? 2 : 3);
      expect(articles.filter((article) => !article.querySelector('.raw-content'))).toEqual([]);
    });

    return container;
  }

  /** the words outside comment text that no div, span or section with data-nosnippet holds */
  function uncoveredWords(container: Element) {
    const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT);
    const uncovered: string[] = [];
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      const parent = node.parentElement!;
      if (!node.textContent?.trim() || parent.closest('.raw-content:not([data-nosnippet])')) {
        continue;
      }
      const wrapper = parent.closest('[data-nosnippet]');
      if (!wrapper || !['DIV', 'SPAN', 'SECTION'].includes(wrapper.tagName)) {
        uncovered.push(node.textContent.trim());
      }
    }
    return uncovered;
  }

  it.each([
    ['signed out', null],
    ['signed in', reader],
  ])("keeps the widget's own words, and only those, out of search snippets, %s", async (_, user) => {
    const container = await renderThreads(user);

    // the comment text stays visible to snippets, apart from the sentence a deleted comment shows
    for (const text of Array.from(container.querySelectorAll('.raw-content'))) {
      const deleted = text.textContent === 'This comment was deleted';
      expect([text.textContent, Boolean(text.closest('[data-nosnippet]'))]).toEqual([text.textContent, deleted]);
    }

    // every other word is inside an element Google reads data-nosnippet on
    expect(uncoveredWords(container)).toEqual([]);
    // the walk saw the widget's words: names, a date, actions, the sort picker and the footer
    expect(container).toHaveTextContent('name c2');
    expect(container).toHaveTextContent('Reply');
    expect(container).toHaveTextContent('Sort by');
    expect(container).toHaveTextContent('Powered by Remark42');
  });

  it('keeps the "Show more" button a phone gets out of search snippets', async () => {
    mockMobile = true;
    try {
      const container = await renderThreads(null, { more: 12 });

      expect(container).toHaveTextContent('Show more');
      expect(uncoveredWords(container)).toEqual([]);
    } finally {
      mockMobile = false;
    }
  });

  it('leaves no empty list behind a thread whose only reply is hidden', async () => {
    const container = await renderThreads(null, { hideReply: true });

    expect(container.querySelectorAll('article')).toHaveLength(2);
    for (const list of Array.from(container.querySelectorAll('[role="list"]'))) {
      expect(list.children.length).toBeGreaterThan(0);
    }
  });

  it('shows no empty list on a phone set to show no comments until asked', async () => {
    jest.spyOn(api, 'getUser').mockResolvedValue(null);
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
    mockMobile = true;
    mockMaxShown = 0;
    try {
      const { container } = render(<ConnectedRoot />, {
        ...stateStub,
        comments: { ...stateStub.comments!, topComments: ['c1'], allComments: {} },
      });

      await waitFor(() => expect(container).toHaveTextContent('Show more'));
      expect(container.querySelector('[role="list"]')).toBeNull();
    } finally {
      mockMobile = false;
      mockMaxShown = undefined;
    }
  });

  it('nests every thread as an item of a list, with a named control that expands it', async () => {
    const container = await renderThreads(null);

    const items = container.querySelectorAll('[role="listitem"]');
    expect(items).toHaveLength(3);
    for (const item of Array.from(items)) {
      expect(item.parentElement).toHaveAttribute('role', 'list');
      // the comment and the control that collapses it, and its replies when it has some
      for (const child of Array.from(item.children)) {
        expect(child.getAttribute('role') ?? child.tagName).toMatch(/^(ARTICLE|list|button)$/);
      }
    }
    for (const list of Array.from(container.querySelectorAll('[role="list"]'))) {
      for (const child of Array.from(list.children)) {
        expect(child).toHaveAttribute('role', 'listitem');
      }
    }

    const controls = container.querySelectorAll('[role="listitem"] > [role="button"]');
    expect(controls).toHaveLength(3);
    for (const control of Array.from(controls)) {
      expect(control).toHaveAttribute('aria-label', 'Collapse or expand thread');
      expect(control).toHaveAttribute('aria-expanded', 'true');
    }
    expect(container.querySelector('[role="listitem"][aria-expanded]')).toBeNull();
  });
});
