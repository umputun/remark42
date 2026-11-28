import { h } from 'preact';
import { act, fireEvent, waitFor } from '@testing-library/preact';

import * as api from 'common/api';
import type { Comment } from 'common/types';
import { render } from 'tests/utils';
import * as pq from 'utils/parse-query';
import * as postMessage from 'utils/post-message';
import * as reader from 'utils/reader';
import * as email from 'utils/email';

import { Counter } from './components/counter';
import { Profile } from './profile';

const someone = { id: 'github_1', name: 'someone', picture: 'http://localhost/someone.png' };

function comment(n: number): Comment {
  return {
    id: `c${n}`,
    pid: n % 2 ? '' : 'parent',
    text: `<p>comment ${n}</p>`,
    title: n % 3 ? `Post ${n}` : undefined,
    locator: { site: 'remark', url: `http://localhost/post/${n}` },
    score: n - 1,
    vote: 0,
    voted_ips: [],
    time: '2026-11-21T15:04:00.000Z',
    user: { ...someone, ip: '', admin: false, block: false, verified: false },
  };
}

/** The profile's HTML, with the dates of its comments, which follow the time zone, left out. */
function markup(container: Element) {
  return container.innerHTML.replace(/(<time[^>]*>)[^<]*(<\/time>)/g, '$1TIME$2');
}

/** the page opens the profile of whoever these are; with a token, the widget frame answers */
function open(query: Record<string, string>, readerIs: typeof someone | null | 'never' = 'never') {
  jest.spyOn(pq, 'parseQuery').mockImplementation(() => query);
  if (readerIs !== 'never') {
    jest.spyOn(reader, 'askReader').mockImplementation(async () => readerIs);
  }
}

function comments(list: Comment[], count = list.length) {
  return jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments: list, count }));
}

beforeEach(() => {
  jest.spyOn(postMessage, 'postMessageToParent').mockReturnValue(true);
});
afterEach(() => {
  jest.restoreAllMocks();
});

// the markup the preact components drew, recorded before they were replaced; innerHTML keeps the
// attributes in the order they were written
describe('the profile draws the markup of the preact components it replaces', () => {
  it('of someone else, loading', async () => {
    open(someone);
    jest.spyOn(api, 'getUserComments').mockImplementation(() => new Promise(() => {}));
    const { container } = render(<Profile />);
    await act(async () => {});
    expect(markup(container)).toMatchSnapshot();
  });

  it('of someone else, with no comments', async () => {
    open(someone);
    comments([]);
    const { container, findByText } = render(<Profile />);
    await findByText('No comments yet.');
    expect(markup(container)).toMatchSnapshot();
  });

  it('of someone else, with comments', async () => {
    open({ ...someone, theme: 'dark' });
    comments([comment(1), comment(2), comment(3)]);
    const { container, findByText } = render(<Profile />);
    await findByText('Comments');
    expect(markup(container)).toMatchSnapshot();
  });

  it('of someone else, with more to load, loading them, and loaded', async () => {
    open(someone);
    const first = Array.from({ length: 10 }, (_, i) => comment(i + 1));
    const getUserComments = comments(first, 12);
    const { container, findByText, getByText } = render(<Profile />);
    await findByText('Load more');
    expect(markup(container)).toMatchSnapshot('more to load');

    let answer: (value: { comments: Comment[]; count: number }) => void = () => {};
    getUserComments.mockImplementation(() => new Promise((resolve) => (answer = resolve)));
    fireEvent.click(getByText('Load more'));
    await act(async () => {});
    expect(markup(container)).toMatchSnapshot('loading more');

    await act(async () => answer({ comments: [comment(11), comment(12)], count: 12 }));
    expect(markup(container)).toMatchSnapshot('all loaded');
  });

  it('of someone else, when the comments fail to load, and on retry', async () => {
    open(someone);
    const getUserComments = jest.spyOn(api, 'getUserComments').mockImplementation(() => Promise.reject(new Error()));
    const { container, findByText, getByText } = render(<Profile />);
    await findByText('Retry');
    expect(markup(container)).toMatchSnapshot('failed');

    getUserComments.mockImplementation(() => new Promise(() => {}));
    fireEvent.click(getByText('Retry'));
    await act(async () => {});
    expect(markup(container)).toMatchSnapshot('retrying');
  });

  it('of someone else who turns out to be the reader', async () => {
    open({ ...someone, token: 't' }, someone);
    comments([comment(1)]);
    const { container, findByText } = render(<Profile />);
    await findByText('Your comments');
    expect(markup(container)).toMatchSnapshot();
  });

  it('own, while the widget frame has not answered', async () => {
    open({ token: 't' });
    jest.spyOn(reader, 'askReader').mockImplementation(() => new Promise(() => {}));
    comments([]);
    const { container } = render(<Profile />);
    await act(async () => {});
    expect(markup(container)).toMatchSnapshot();
  });

  it('own, with no comments, signed in with email', async () => {
    const reader = { id: 'email_abc', name: 'me', picture: '' };
    open({ token: 't' }, reader);
    comments([]);
    const { container, findByText } = render(<Profile />);
    await findByText('Edit avatar');
    await findByText('No comments yet.');
    expect(markup(container)).toMatchSnapshot();
  });

  it('own, as an anonymous reader', async () => {
    const reader = { id: 'anonymous_abc', name: 'anon', picture: 'http://localhost/anon.png' };
    open({ token: 't' }, reader);
    comments([comment(1)]);
    const { container, findByText } = render(<Profile />);
    await findByText('Your comments');
    expect(markup(container)).toMatchSnapshot();
  });

  it('own, signing out', async () => {
    open({ ...someone, token: 't' }, someone);
    comments([]);
    const { container, findByTitle } = render(<Profile />);
    fireEvent.click(await findByTitle('Sign out'));
    expect(markup(container)).toMatchSnapshot();
  });

  it('own, asking for the data to be deleted', async () => {
    jest.spyOn(email, 'requestDeletion').mockImplementation(() => new Promise(() => {}));
    open({ ...someone, token: 't' }, someone);
    comments([]);
    const { container, findByText } = render(<Profile />);
    fireEvent.click(await findByText('Request deletion of my data'));
    await act(async () => {});
    expect(markup(container)).toMatchSnapshot();
  });

  it('closing', async () => {
    open(someone);
    comments([]);
    const { container, findByText, getByTitle } = render(<Profile />);
    await findByText('No comments yet.');
    fireEvent.click(getByTitle('Close profile'));
    expect(markup(container)).toMatchSnapshot();
  });

  it('of nobody, once the widget frame says nobody is signed in', async () => {
    open({ token: 't' }, null);
    comments([]);
    const { container } = render(<Profile />);
    await waitFor(() => expect(postMessage.postMessageToParent).toHaveBeenCalled());
    expect(markup(container)).toMatchSnapshot();
  });

  it('the counter', () => {
    expect(render(<Counter>{42}</Counter>).container.innerHTML).toMatchSnapshot();
  });
});
