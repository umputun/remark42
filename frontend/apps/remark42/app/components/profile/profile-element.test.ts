import '@testing-library/jest-dom';
import { fireEvent, waitFor } from '@testing-library/preact';

import { createIntl } from 'common/intl';
import en from 'locales/en.json';
import type { Comment } from 'common/types';
import * as pq from 'utils/parse-query';
import * as postMessage from 'utils/post-message';
import * as reader from 'utils/reader';

import type { ProfileDeps } from './profile-element.js';
import { mountProfile } from './profile-element.js';

const intl = createIntl('en', en);
const someone = { id: 'github_1', name: 'someone', picture: '' };

function comment(n: number): Comment {
  return {
    id: `c${n}`,
    pid: '',
    text: `<p>comment ${n}</p>`,
    locator: { site: 'remark', url: 'http://localhost/post' },
    score: 0,
    vote: 0,
    voted_ips: [],
    time: '2026-11-21T15:04:00.000Z',
    user: { ...someone, ip: '', admin: false, block: false, verified: false },
  };
}

function deps(change: Partial<ProfileDeps> = {}): ProfileDeps {
  return {
    intl,
    getUserComments: () => new Promise(() => {}),
    requestDeletion: () => Promise.resolve(),
    votes: { intl, putVote: () => new Promise(() => {}), patch: jest.fn(), positiveScore: () => false },
    ...change,
  };
}

const mounted: Array<ReturnType<typeof mountProfile>> = [];

/** the profile after an empty text node in a container on the page, as the wrapper places it */
function mount(d: ProfileDeps) {
  const container = document.createElement('div');
  const anchor = document.createTextNode('');
  container.append(anchor);
  document.body.append(container);
  const profile = mountProfile(anchor, d);
  mounted.push(profile);
  return { container, profile };
}

const settle = () => new Promise((resolve) => setTimeout(resolve));

beforeEach(() => {
  jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...someone }));
  jest.spyOn(postMessage, 'postMessageToParent').mockReturnValue(true);
});
afterEach(() => {
  mounted.splice(0).forEach((profile) => profile.destroy());
  document.body.innerHTML = '';
  jest.restoreAllMocks();
});

describe('mountProfile', () => {
  it('keeps the comments it drew when it loads more', async () => {
    const getUserComments = jest
      .fn()
      .mockResolvedValueOnce({ comments: Array.from({ length: 10 }, (_, i) => comment(i)), count: 11 })
      .mockResolvedValueOnce({ comments: [comment(10)], count: 11 });
    const { container } = mount(deps({ getUserComments }));
    await settle();
    const before = Array.from(container.querySelectorAll('article'));
    const nodes = Array.from(container.querySelectorAll('article *'));

    fireEvent.click(container.querySelector('.loadMoreWrapper button')!);
    await settle();

    const after = Array.from(container.querySelectorAll('article'));
    expect(after).toHaveLength(11);
    before.forEach((article, i) => expect(after[i]).toBe(article));
    const kept = Array.from(container.querySelectorAll('article *')).slice(0, nodes.length);
    kept.forEach((node, i) => expect(node).toBe(nodes[i]));
    expect(getUserComments).toHaveBeenLastCalledWith('github_1', { skip: 10, limit: 10 });
  });

  it('asks for more through the deps of the latest update', async () => {
    const first = jest
      .fn()
      .mockResolvedValue({ comments: Array.from({ length: 10 }, (_, i) => comment(i)), count: 20 });
    const second = jest.fn(() => new Promise<never>(() => {}));
    const { container, profile } = mount(deps({ getUserComments: first }));
    await settle();

    profile.update(deps({ getUserComments: second }));
    fireEvent.click(container.querySelector('.loadMoreWrapper button')!);

    expect(first).toHaveBeenCalledTimes(1);
    expect(second).toHaveBeenCalledTimes(1);
  });

  it('fades in once it is on the page, not before', async () => {
    const records: MutationRecord[] = [];
    const container = document.createElement('div');
    const anchor = document.createTextNode('');
    container.append(anchor);
    document.body.append(container);
    const observer = new MutationObserver((list) => records.push(...list));
    observer.observe(container, { childList: true, attributes: true, subtree: true, attributeOldValue: true });

    mounted.push(mountProfile(anchor, deps()));
    await settle();
    observer.disconnect();

    const root = container.lastChild as HTMLElement;
    const added = records.findIndex((r) => r.type === 'childList' && r.addedNodes[0] === root);
    // the class changes on the page, from what the profile was inserted with
    const appear = records.findIndex(
      (r) => r.type === 'attributes' && r.target === root && r.attributeName === 'class'
    );
    expect(added).toBeGreaterThanOrEqual(0);
    expect(appear).toBeGreaterThan(added);
    expect(records[appear].oldValue).toBe('profile root');
    expect(root).toHaveClass('rootAppear');
  });

  it('draws nothing of an answer that comes after it is gone, and listens no more', async () => {
    let answer: (value: { comments: Comment[]; count: number }) => void = () => {};
    const { container, profile } = mount(deps({ getUserComments: () => new Promise((resolve) => (answer = resolve)) }));

    const root = container.querySelector('.profile')!;
    profile.destroy();
    answer({ comments: [comment(1)], count: 1 });
    await settle();
    fireEvent.keyDown(document, { code: 'Escape' });

    expect(container.children).toHaveLength(0);
    expect(root.querySelector('article')).toBeNull();
    expect(postMessage.postMessageToParent).not.toHaveBeenCalled();
  });

  it('closes on Escape, and from the button once its transition ends', async () => {
    const { container } = mount(deps());

    fireEvent.keyDown(document, { code: 'Escape' });
    expect(postMessage.postMessageToParent).toHaveBeenCalledWith({ profile: null });

    jest.mocked(postMessage.postMessageToParent).mockClear();
    fireEvent.click(container.querySelector('.closeButtonWrapper')!);
    const root = container.querySelector('.profile')!;
    expect(root).toHaveClass('rootDisappear');
    expect(root).not.toHaveClass('rootAppear');
    expect(postMessage.postMessageToParent).not.toHaveBeenCalled();
    // jsdom has no TransitionEvent, which the testing library's helper needs
    root.dispatchEvent(new Event('transitionend'));
    expect(postMessage.postMessageToParent).toHaveBeenCalledWith({ profile: null });
  });

  it('shows the reader the widget frame answers for, then their comments', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ token: 't' }));
    jest.spyOn(reader, 'askReader').mockImplementation(async () => someone);
    const getUserComments = jest.fn().mockResolvedValue({ comments: [comment(1)], count: 1 });
    const { container } = mount(deps({ getUserComments }));
    expect(container.querySelector('.profile')).toBeNull();

    await waitFor(() => expect(container.querySelector('.profile-title')).toHaveTextContent('someone'));
    await waitFor(() => expect(container.querySelector('h3')).toHaveTextContent('Your comments'));
    expect(getUserComments).toHaveBeenCalledTimes(1);
    expect(container.querySelector('.profile')).toHaveClass('rootAppear');
  });
});
