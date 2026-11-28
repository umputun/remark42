import '@testing-library/jest-dom';
import { fireEvent, screen, waitFor } from '@testing-library/preact';

import { render } from 'tests/utils';
import * as api from 'common/api';
import * as postMessage from 'utils/post-message';
import * as authApi from 'components/auth/auth.api';
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
    page: { countLeft: 0, lastComment: '', loaded: 0, undeleted: 0 },
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

  it('signs the reader out when the profile asks for it', async () => {
    jest.spyOn(api, 'getUser').mockImplementation(async () => null);
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
    jest.spyOn(postMessage, 'updateIframeHeight').mockImplementation(() => undefined);
    const logout = jest.spyOn(authApi, 'logout').mockImplementation(async () => undefined);

    render(<ConnectedRoot />, stateStub);

    // the page passes the profile's message on. The profile has ended nothing: its frame is
    // removed as it closes, so the logout request has to come from this document
    window.dispatchEvent(new MessageEvent('message', { data: { signout: true }, source: window.parent }));

    // the roots of the tests above still listen, so the count is not one
    await waitFor(() => expect(logout).toHaveBeenCalled());
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
  async function renderThreads(user: User | null, { more = 0, hideReply = false, countLeft = 0 } = {}) {
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
        page: { countLeft, lastComment: 'c3', loaded: 3, undeleted: 3 },
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

  it('loads the next page on a desktop when the server has more comments than it sent', async () => {
    const container = await renderThreads(null, { countLeft: 5 });
    const getPostComments = jest.mocked(api.getPostComments);
    expect(getPostComments).toHaveBeenCalledTimes(1);
    // the page is held back until the button has been seen disabled, and then answered, since the
    // next load waits for this one
    let answer!: () => void;
    getPostComments.mockImplementation(
      () =>
        new Promise((resolve) => {
          answer = () => resolve({ comments: [], info: { url: 'test-url', count: 0 } });
        })
    );

    const button = Array.from(container.querySelectorAll('button')).find((b) => b.textContent === 'Show more');
    expect(button).toBeDefined();
    button!.click();

    await waitFor(() => expect(getPostComments).toHaveBeenLastCalledWith('-active', { limit: 100, offsetId: 'c3' }));
    // the page is still on its way, so a second click cannot ask for it again
    await waitFor(() => expect(button).toBeDisabled());
    answer();
  });

  it('shows no "Show more" on a desktop when the whole thread is loaded', async () => {
    const container = await renderThreads(null);

    expect(container).not.toHaveTextContent('Show more');
  });

  /** how many top-level threads the widget renders: the items of the list of threads */
  function shownThreads(container: Element) {
    return container.querySelector('[role="list"]')?.children.length ?? 0;
  }

  it.each([
    ['a desktop', false],
    ['a phone', true],
  ])('limits the top-level comments to max_shown_comments on %s and shows ten more on demand', async (_, mobile) => {
    mockMobile = mobile;
    mockMaxShown = 2;
    try {
      // the two threads of the fixture and 13 more, the whole thread loaded
      const container = await renderThreads(null, { more: 13 });

      expect(shownThreads(container)).toBe(2);
      fireEvent.click(screen.getByRole('button', { name: 'Show more' }));
      expect(shownThreads(container)).toBe(12);
      fireEvent.click(screen.getByRole('button', { name: 'Show more' }));
      expect(shownThreads(container)).toBe(15);
      expect(screen.queryByRole('button', { name: 'Show more' })).not.toBeInTheDocument();
      // every thread was loaded, so the button revealed them and asked the server for nothing
      expect(api.getPostComments).toHaveBeenCalledTimes(1);
    } finally {
      mockMobile = false;
      mockMaxShown = undefined;
    }
  });

  it('shows the first ten top-level comments on a phone when the site sets no max_shown_comments', async () => {
    mockMobile = true;
    try {
      const container = await renderThreads(null, { more: 12 });

      expect(shownThreads(container)).toBe(10);
      expect(screen.getByRole('button', { name: 'Show more' })).toBeInTheDocument();
    } finally {
      mockMobile = false;
    }
  });

  it('shows every top-level comment on a desktop when the site sets no max_shown_comments', async () => {
    const container = await renderThreads(null, { more: 12 });

    expect(shownThreads(container)).toBe(14);
    expect(screen.queryByRole('button', { name: 'Show more' })).not.toBeInTheDocument();
  });

  it('leaves no empty list behind a thread whose only reply is hidden', async () => {
    const container = await renderThreads(null, { hideReply: true });

    expect(container.querySelectorAll('article')).toHaveLength(2);
    for (const list of Array.from(container.querySelectorAll('[role="list"]'))) {
      expect(list.children.length).toBeGreaterThan(0);
    }
  });

  // max_shown_comments is at least 1, so the list is empty only when a loaded page held no thread to
  // show, every one by a user the reader hid, and the server has more
  it('shows no empty list when a loaded page left no thread to show and the server has more', async () => {
    jest.spyOn(api, 'getUser').mockResolvedValue(null);
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
    const { container } = render(<ConnectedRoot />, {
      ...stateStub,
      comments: { ...stateStub.comments!, page: { countLeft: 3, lastComment: 'hidden', loaded: 2, undeleted: 2 } },
    });

    await waitFor(() => expect(container).toHaveTextContent('Show more'));
    expect(container.querySelector('[role="list"]')).toBeNull();
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

  describe('opening a link to a comment', () => {
    const ids = ['first', 'second', 'third'];
    const comment = (id: string): Comment => ({
      id,
      pid: '',
      text: `text of ${id}`,
      time: new Date().toString(),
      score: 0,
      vote: 0,
      voted_ips: [],
      locator: { url: 'test-url', site: 'remark' },
      user: { id: 'someone', name: 'someone', picture: '', ip: '', admin: false, block: false, verified: false },
    });
    const state: Partial<StoreState> = {
      ...stateStub,
      comments: {
        ...stateStub.comments!,
        topComments: ids,
        allComments: Object.fromEntries(ids.map((id) => [id, comment(id)])),
      },
      info: { url: 'test-url', count: ids.length, read_only: false },
    };
    const { IntersectionObserver } = window;

    beforeEach(() => {
      // the widget is off screen, as it is below a post, so no comment is ever reported in view
      window.IntersectionObserver = class {
        observe() {
          // never in view
        }
        unobserve() {
          // never in view
        }
        disconnect() {
          // never in view
        }
      } as unknown as typeof window.IntersectionObserver;
      // replaceState fires no hashchange, which would reach the listeners of earlier tests' widgets
      window.history.replaceState(null, '', '#remark42__comment-third');
    });

    afterEach(() => {
      window.IntersectionObserver = IntersectionObserver;
      window.history.replaceState(null, '', window.location.pathname);
    });

    it('loads the rest of the thread when the linked comment is on a page not loaded yet', async () => {
      jest.spyOn(api, 'getUser').mockResolvedValue(null);
      const getPostComments = jest.spyOn(api, 'getPostComments').mockResolvedValue({ comments: [], info: state.info! });
      window.history.replaceState(null, '', '#remark42__comment-fourth');

      render(<ConnectedRoot />, {
        ...state,
        comments: { ...state.comments!, page: { countLeft: 2, lastComment: 'third', loaded: 3, undeleted: 3 } },
      });

      await waitFor(() => expect(getPostComments).toHaveBeenCalledWith('-active', { limit: 0, offsetId: 'third' }));
    });

    it('renders the linked comment and reports the height before asking the page to scroll to it', async () => {
      jest.spyOn(api, 'getUser').mockResolvedValue(null);
      jest.spyOn(api, 'getPostComments').mockResolvedValue({ comments: [], info: state.info! });
      // a screen of two placeholders' worth, so the comments around the linked one cover the thread
      jest.spyOn(window.screen, 'height', 'get').mockReturnValue(100);
      // a comment out of view is an empty placeholder of a guessed size
      const isRendered = (id: string) => !!document.getElementById(`remark42__comment-${id}`)?.childElementCount;
      const rendered = () => ids.filter(isRendered).join(', ');
      // jsdom lays nothing out, so the linked comment's offset is made to depend on the comments
      // above it: 100 px for a placeholder, 200 px for a rendered comment
      const { getBoundingClientRect } = Element.prototype;
      jest.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
        if (this.id !== 'remark42__comment-third') {
          return getBoundingClientRect.call(this);
        }
        return { top: ['first', 'second'].reduce((top, id) => top + (isRendered(id) ? 200 : 100), 0) } as DOMRect;
      });
      const sent: string[] = [];
      jest.spyOn(postMessage, 'updateIframeHeight').mockImplementation(() => {
        sent.push(`height with ${rendered()}`);
      });
      jest.spyOn(postMessage, 'postMessageToParent').mockImplementation((data) => {
        if (typeof data.scrollTo === 'number') {
          sent.push(`scrollTo ${data.scrollTo} with ${rendered()}`);
        }
        return true;
      });

      render(<ConnectedRoot />, state);

      await waitFor(() => expect(sent.some((message) => message.startsWith('scrollTo'))).toBe(true));
      // the frame reports its height only while on screen, so the one sent here is the one the
      // page has when it scrolls. The comments above count too: the nearest touches the top of the
      // screen once the page has scrolled, so as a placeholder it would grow and push this one down
      expect(sent.slice(-2)).toEqual(['height with first, second, third', 'scrollTo 400 with first, second, third']);
    });
  });

  it('looks the comment of the url hash up by its id and never uses the hash as a selector', async () => {
    jest.spyOn(api, 'getUser').mockResolvedValue(null);
    jest.spyOn(api, 'getPostComments').mockResolvedValue({ comments: [], info: stateStub.info! });
    jest.spyOn(postMessage, 'updateIframeHeight').mockImplementation(() => undefined);
    const postMessageToParent = jest.spyOn(postMessage, 'postMessageToParent').mockImplementation(() => true);
    const addEventListener = jest.spyOn(window, 'addEventListener');
    const querySelector = jest.spyOn(document, 'querySelector');

    const comment = document.createElement('article');
    comment.id = 'remark42__comment-1';
    document.body.append(comment);

    try {
      render(<ConnectedRoot />, stateStub);
      await waitFor(() => expect(addEventListener).toHaveBeenCalledWith('hashchange', expect.any(Function)));

      const changeHash = (hash: string) =>
        window.dispatchEvent(new HashChangeEvent('hashchange', { newURL: `http://localhost/${hash}` }));

      // the embedding page sends the hash and gets `scrollTo` back when it matches, so a hash
      // read as a selector list would tell that page what the widget shows to the signed-in user
      changeHash('#remark42__comment-,article');
      // a hash with no comment on screen is looked up once more after the hidden ones are shown
      await new Promise((resolve) => setTimeout(resolve, 600));
      expect(postMessageToParent).not.toHaveBeenCalled();

      changeHash('#remark42__comment-1');
      await waitFor(() => expect(postMessageToParent).toHaveBeenCalledWith({ scrollTo: expect.any(Number) }));
      expect(postMessageToParent).toHaveBeenCalledTimes(1);

      // the first lookup decides whether hidden comments are expanded, which changes the height the
      // widget posts to the page, so neither lookup may read the hash as a selector
      const hashQueries = querySelector.mock.calls.filter(([selector]) => selector.startsWith('#remark42__comment-'));
      expect(hashQueries).toEqual([]);
    } finally {
      comment.remove();
    }
  });
});
