import '@testing-library/jest-dom';
import { fireEvent } from '@testing-library/preact';

import { createIntl } from 'common/intl';
import en from 'locales/en.json';
import de from 'locales/de.json';
import { StaticStore } from 'common/static-store';
import { CommentMode } from 'common/types';
import * as copy from 'common/copy';
import * as countdownModule from 'components/countdown.js';
import * as postMessage from 'utils/post-message';
import { draw } from 'tests/plain';
import { signInDeps } from 'components/auth/auth';
import { subscribeByEmailArgs } from 'components/comment-form/__subscribe-by-email/comment-form__subscribe-by-email';
import { subscribeByTelegramArgs } from 'components/comment-form/__subscribe-by-telegram/comment-form__subscribe-by-telegram';

import type { CommentDeps, CommentProps } from './comment-element.js';
import { mountComment } from './comment-element.js';

const NOW = Date.parse('2026-11-21T15:05:00.000Z');
const intl = createIntl('en', en);

function props(change: Partial<CommentProps> = {}): CommentProps {
  return {
    view: 'main',
    theme: 'light',
    level: 0,
    repliesCount: 0,
    post_info: { url: 'http://localhost/post/1', count: 2, read_only: false },
    user: { id: 'reader', name: 'reader', picture: '', ip: '', admin: false, block: false, verified: false },
    data: {
      id: 'c1',
      pid: 'p1',
      text: '<p>text</p>',
      orig: 'text',
      vote: 0,
      score: 2,
      voted_ips: [],
      time: new Date(NOW - 60_000).toISOString(),
      locator: { url: 'http://localhost/post/1', site: 'remark' },
      user: {
        id: 'author',
        name: 'author',
        picture: 'http://localhost/a.png',
        ip: '',
        admin: false,
        block: false,
        verified: false,
      },
    },
    ...change,
  };
}

/** the reader wrote the comment */
function own(change: Partial<CommentProps> = {}) {
  const p = props(change);
  return { ...p, data: { ...p.data, user: { ...p.user! } } };
}

let textareas = 0;

function formBase(): NonNullable<CommentDeps['form']> {
  return {
    nextTextareaId: () => `textarea_${++textareas}`,
    loadEmoji: () => Promise.resolve({}),
    theme: 'light',
    signIn: signInDeps(intl, 'light', jest.fn() as never),
    rssIntl: intl,
    email: subscribeByEmailArgs(null, 'light', intl, jest.fn()),
    telegram: subscribeByTelegramArgs(null, 'light', intl),
  };
}

function deps(change: Partial<CommentDeps> = {}): CommentDeps {
  return {
    intl,
    partsIntl: intl,
    votes: { intl, putVote: () => new Promise(() => {}), patch: jest.fn(), positiveScore: () => false },
    actions: { setReplyEditState: jest.fn() },
    getPreview: () => Promise.resolve(''),
    ...change,
  };
}

function mount(p: CommentProps = props(), d: CommentDeps = deps()) {
  const { root, part } = draw('article', (article) => mountComment(article, p, d));
  return { root, comment: part };
}

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Node[], before: Node[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

const all = (root: Element) => Array.from(root.querySelectorAll('*'));
const button = (root: Element, text: string) =>
  Array.from(root.querySelectorAll('button')).find((b) => b.textContent === text)!;

beforeEach(() => {
  jest.spyOn(Date, 'now').mockImplementation(() => NOW);
});
afterEach(() => {
  textareas = 0;
  localStorage.clear();
  jest.restoreAllMocks();
  jest.useRealTimers();
});

describe('mountComment', () => {
  it('keeps every node across an update that changes what they show', () => {
    const p = props();
    const { root, comment } = mount(p);
    const before = all(root);

    comment.update({ ...p, data: { ...p.data, score: 5, vote: 1 } }, deps());

    expectSameNodes(all(root), before);
    expect(root.querySelector('[title="Score"]')).toHaveTextContent('5');
  });

  it('writes the text again only when it changes, so its images are not reloaded', () => {
    const p = props();
    const { root, comment } = mount(p);
    const paragraph = root.querySelector('.raw-content p');

    comment.update({ ...p, data: { ...p.data, score: 9 } }, deps());
    expect(root.querySelector('.raw-content p')).toBe(paragraph);

    comment.update({ ...p, data: { ...p.data, text: '<p>new</p>' } }, deps());
    expect(root.querySelector('.raw-content')).toHaveTextContent('new');
  });

  it('calls the handlers and the locale of the latest update', () => {
    const p = props();
    const first = deps();
    const { root, comment } = mount(p, first);
    const second = deps({ actions: { setReplyEditState: jest.fn() } });
    comment.update(p, second);

    fireEvent.click(button(root, 'Reply'));
    expect(first.actions.setReplyEditState).not.toHaveBeenCalled();
    expect(second.actions.setReplyEditState).toHaveBeenCalledWith({ id: 'c1', state: CommentMode.Reply });

    comment.update({ ...p, user: { ...p.user!, admin: true } }, deps({ intl: createIntl('de', de) }));
    expect(root.querySelector('.verificationButton')).toHaveAttribute('title', de['comment.toggle-verification']);
  });

  it('keeps the reply form, with its text, focus and caret, while the comment redraws', () => {
    const p = props({ editMode: CommentMode.Reply });
    const { root, comment } = mount(p, deps({ form: formBase() }));
    const field = root.querySelector('textarea')!;
    field.focus();
    field.value = 'half a reply';
    field.setSelectionRange(4, 6);
    fireEvent.input(field);

    comment.update({ ...p, data: { ...p.data, score: 7 } }, deps({ form: formBase() }));
    comment.update({ ...p, theme: 'dark' }, deps({ form: formBase() }));

    expect(root.querySelector('textarea')).toBe(field);
    expect(field.value).toBe('half a reply');
    expect([field.selectionStart, field.selectionEnd]).toEqual([4, 6]);
    expect(document.activeElement).toBe(field);
    expect(field.id).toBe('textarea_1');
  });

  it('gives an edit its own form with the comment’s text, and takes the form away when it closes', () => {
    const p = own({ editMode: CommentMode.Reply });
    const { root, comment } = mount(p, deps({ form: formBase() }));
    const reply = root.querySelector('form');

    comment.update({ ...p, editMode: CommentMode.Edit }, deps({ form: formBase() }));
    const edit = root.querySelector('form');
    expect(edit).not.toBe(reply);
    expect(root.querySelector('textarea')).toHaveValue('text');
    expect(root.querySelector('textarea')!.id).toBe('textarea_2');

    comment.update({ ...p, editMode: CommentMode.None }, deps({ form: formBase() }));
    expect(root.querySelector('form')).toBeNull();
  });

  it('mounts a form once it is on the page, so its field is sized to the text it opens with', () => {
    // jsdom measures nothing, so a field is as tall as its text only while it is on the page
    jest.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockImplementation(function (this: HTMLElement) {
      return this.isConnected ? 120 : 0;
    });
    const { root } = mount(own({ editMode: CommentMode.Edit }), deps({ form: formBase() }));

    expect(root.querySelector('textarea')!.style.height).toBe('120px');
  });

  it('leaves a class the thread put on it through a redraw, until its own class changes', () => {
    const p = props();
    const { root, comment } = mount(p);
    root.classList.add('comment_highlighting');

    comment.update({ ...p, data: { ...p.data, score: 4 } }, deps());
    expect(root).toHaveClass('comment_highlighting');

    comment.update({ ...p, theme: 'dark' }, deps());
    expect(root).not.toHaveClass('comment_highlighting');
  });

  it('draws a placeholder with no class after it was drawn in full', () => {
    const p = props({ inView: false, editMode: CommentMode.Reply });
    const { root, comment } = mount(p);
    expect(root).toHaveClass('root');

    comment.update({ ...p, editMode: CommentMode.None }, deps());
    expect(root.hasAttribute('class')).toBe(false);
    expect(root.children).toHaveLength(0);

    comment.update({ ...p, editMode: CommentMode.None, inView: true }, deps());
    expect(root).toHaveClass('root');
  });

  it('draws no form without the form deps, nor in the pinned block', () => {
    const p = props({ editMode: CommentMode.Reply });
    expect(mount(p).root.querySelector('form')).toBeNull();
    expect(mount({ ...p, view: 'pinned' }, deps({ form: formBase() })).root.querySelector('form')).toBeNull();
  });

  it('closes a sent reply, and an edit once it is saved', async () => {
    const addComment = jest.fn(() => Promise.resolve());
    const updateComment = jest.fn(() => Promise.resolve());
    const setReplyEditState = jest.fn();
    const p = own({ editMode: CommentMode.Edit });
    const d = deps({ form: formBase(), actions: { addComment, updateComment, setReplyEditState } });
    const { root } = mount(p, d);
    const field = root.querySelector('textarea')!;
    field.value = 'edited';
    fireEvent.input(field);

    fireEvent.submit(root.querySelector('form')!);
    await Promise.resolve();
    await Promise.resolve();

    expect(updateComment).toHaveBeenCalledWith('c1', 'edited');
    expect(setReplyEditState).toHaveBeenLastCalledWith({ id: 'c1', state: CommentMode.None });
  });

  it('keeps the text and its nodes through collapsing and expanding', () => {
    const p = props();
    const { root, comment } = mount(p);
    const text = root.querySelector('.raw-content');
    const avatar = root.querySelector('img');

    comment.update({ ...p, collapsed: true }, deps());
    expect(root.querySelector('.raw-content')).toBeNull();
    expect(root.querySelector('img')).toBeNull();

    comment.update({ ...p, collapsed: false }, deps());
    expect(root.querySelector('.raw-content')).toBe(text);
    expect(root.querySelector('img')).toBe(avatar);
  });

  describe('images it does not show', () => {
    let sources: string[];
    beforeEach(() => {
      sources = [];
      const descriptor = Object.getOwnPropertyDescriptor(HTMLImageElement.prototype, 'src')!;
      jest.spyOn(HTMLImageElement.prototype, 'src', 'set').mockImplementation(function (this: HTMLImageElement, v) {
        sources.push(v);
        descriptor.set!.call(this, v);
      });
    });

    it.each<[string, Partial<CommentProps>]>([
      ['out of view', { inView: false }],
      ['collapsed', { collapsed: true }],
      ['in a profile', { view: 'user' }],
    ])('loads no avatar while %s', (_, change) => {
      mount(props(change));
      expect(sources).not.toContain('http://localhost/a.png');
    });

    it('loads the Patreon badge only for a subscriber', () => {
      const p = props();
      const { comment } = mount(p);
      expect(sources.filter((s) => s.endsWith('patreon.svg'))).toHaveLength(0);

      comment.update({ ...p, data: { ...p.data, user: { ...p.data.user, paid_sub: true } } }, deps());
      comment.update({ ...p, data: { ...p.data, score: 3, user: { ...p.data.user, paid_sub: true } } }, deps());
      expect(sources.filter((s) => s.endsWith('patreon.svg'))).toHaveLength(1);
    });
  });

  it('draws a placeholder of the size the article has until it comes into view', () => {
    const p = props({ inView: false });
    const { root, part: comment } = draw('article', (article) => {
      Object.defineProperty(article, 'scrollWidth', { get: () => 321 });
      Object.defineProperty(article, 'scrollHeight', { get: () => 45 });
      return mountComment(article, p, deps());
    });

    expect(root.style.width).toBe('321px');
    expect(root.style.height).toBe('45px');
    expect(root.children).toHaveLength(0);

    comment.update({ ...p, inView: true }, deps());
    expect(root.getAttribute('style')).toBe('');
    expect(root.querySelector('.raw-content')).toHaveTextContent('text');

    comment.update({ ...p, inView: false }, deps());
    expect(root.querySelector('.raw-content')).toHaveTextContent('text');
  });

  it('takes Edit and Delete away when the edit window ends, and leaves no timer behind', () => {
    jest.useFakeTimers({ now: NOW });
    const p = own();
    p.data.time = new Date(NOW - 298_000).toISOString();
    const { root, comment } = mount(p);
    expect(button(root, 'Edit')).toBeVisible();
    expect(button(root, 'Delete')).toBeVisible();

    jest.advanceTimersByTime(3_000);
    expect(button(root, 'Edit')).toBeUndefined();
    expect(button(root, 'Delete')).toBeUndefined();
    expect(jest.getTimerCount()).toBe(0);

    comment.update(p, deps());
    expect(button(root, 'Delete')).toBeUndefined();
  });

  it('waits out an edit window longer than one timeout can wait', () => {
    jest.useFakeTimers({ now: NOW });
    StaticStore.config.edit_duration = 30 * 24 * 60 * 60;
    const p = own({ post_info: { url: '', count: 1, read_only: true } });
    p.data.time = new Date(NOW).toISOString();
    const setTimeoutSpy = jest.spyOn(window, 'setTimeout');
    const { root } = mount(p);

    // a delay past 2^31-1 ms fires at once in a browser, which the fake clock does not do
    const delays = (setTimeoutSpy.mock.calls as unknown[][]).map((call) => Number(call[1] ?? 0));
    expect(Math.max(...delays)).toBeLessThanOrEqual(2 ** 31 - 1);
    jest.advanceTimersByTime(25 * 24 * 60 * 60 * 1000);
    expect(button(root, 'Delete')).toBeVisible();
    jest.advanceTimersByTime(6 * 24 * 60 * 60 * 1000);
    expect(button(root, 'Delete')).toBeUndefined();
  });

  it('shows Copied for three seconds', async () => {
    jest.useFakeTimers({ now: NOW });
    jest.spyOn(copy, 'copy').mockImplementation(() => Promise.resolve());
    const p = props({ user: { ...props().user!, admin: true } });
    const { root } = mount(p);

    fireEvent.click(button(root, 'Copy'));
    await Promise.resolve();
    await Promise.resolve();
    expect(button(root, 'Copied!')).toBeDisabled();
    expect(copy.copy).toHaveBeenCalledWith(expect.stringMatching(/^author .+\ntext$/));

    jest.advanceTimersByTime(3_000);
    expect(button(root, 'Copy')).toBeEnabled();
  });

  it('asks about a block once the select settles, with the period in the comment’s locale', () => {
    jest.useFakeTimers({ now: NOW });
    const confirm = jest.spyOn(window, 'confirm').mockReturnValue(true);
    const blockUser = jest.fn();
    const p = props({ user: { ...props().user!, admin: true } });
    const { root } = mount(p, deps({ intl: createIntl('de', de), actions: { blockUser } }));
    const select = root.querySelector('select')!;

    fireEvent.change(select, { target: { value: '1440m' } });
    fireEvent.change(select, { target: { value: '43200m' } });
    jest.advanceTimersByTime(99);
    expect(confirm).not.toHaveBeenCalled();
    jest.advanceTimersByTime(1);

    expect(confirm).toHaveBeenCalledTimes(1);
    expect(confirm).toHaveBeenCalledWith('Möchten Sie author wirklich für einen Monat sperren?');
    expect(blockUser).toHaveBeenCalledWith('author', 'author', '43200m');
  });

  it('scrolls the page to the parent comment, or the parent within the frame', () => {
    const parent = document.createElement('div');
    parent.id = 'remark42__comment-p1';
    parent.scrollIntoView = jest.fn();
    document.body.append(parent);
    const post = jest.spyOn(postMessage, 'postMessageToParent').mockReturnValue(true);
    try {
      const { root } = mount(props({ level: 1 }));
      const link = root.querySelector<HTMLAnchorElement>('[title="Go to parent comment"]')!;

      fireEvent.click(link);
      expect(post).toHaveBeenCalledWith({ scrollTo: expect.any(Number) });
      expect(parent.scrollIntoView).not.toHaveBeenCalled();

      post.mockReturnValue(false);
      fireEvent.click(link);
      expect(parent.scrollIntoView).toHaveBeenCalledTimes(1);
    } finally {
      parent.remove();
    }
  });

  it('leaves no timer behind once destroyed, whatever was pending', async () => {
    jest.useFakeTimers({ now: NOW });
    jest.spyOn(copy, 'copy').mockImplementation(() => Promise.resolve());
    const confirm = jest.spyOn(window, 'confirm').mockReturnValue(true);
    const p = own({ user: { ...props().user!, id: 'author', admin: true } });
    const reader = props({ user: { ...props().user!, admin: true } });
    const { root: mine, comment: ownComment } = mount(p);
    const { root, comment } = mount(reader);

    fireEvent.click(button(root, 'Copy'));
    await Promise.resolve();
    await Promise.resolve();
    fireEvent.change(root.querySelector('select')!, { target: { value: '43200m' } });
    expect(mine.querySelector('[role="timer"]')).toBeVisible();
    expect(jest.getTimerCount()).toBeGreaterThan(0);

    comment.destroy();
    ownComment.destroy();

    expect(jest.getTimerCount()).toBe(0);
    jest.advanceTimersByTime(10_000);
    expect(confirm).not.toHaveBeenCalled();
  });

  it('draws once more, after the pass, when a part calls back while it is drawn', () => {
    // a countdown that has already run out when it is given its moment, as one does when the clock
    // passes the deadline between the comment working it out and the actions drawing it
    const made = jest.spyOn(countdownModule, 'countdown').mockImplementation((onTimePassed) => ({
      text: document.createTextNode(''),
      countTo: () => onTimePassed(),
      stop: () => {},
    }));
    const { root } = mount(own());

    // one set of actions, drawn again after the pass: drawn inside it, the comment would make a
    // second set while the first was still being made, and lose one of them
    expect(made).toHaveBeenCalledTimes(1);
    expect(root.querySelectorAll('.comment-actions')).toHaveLength(1);
    expect(button(root, 'Edit')).toBeUndefined();
    expect(button(root, 'Delete')).toBeUndefined();
  });
});
