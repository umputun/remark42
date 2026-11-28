import { h } from 'preact';
import { act, fireEvent, render } from '@testing-library/preact';

import * as api from 'common/api';
import { IntlProvider, useIntl } from 'common/intl';
import { CommentMode } from 'common/types';
import en from 'locales/en.json';
import { mockStore } from '__stubs__/store';
import { Provider } from 'store/context';
import { COMMENT_PATCH } from 'store/comments/types';
import { CommentForm } from 'components/comment-form';
import * as email from 'components/comment-form/__subscribe-by-email/comment-form__subscribe-by-email';
import * as emojiData from 'components/comment-form/emoji-data';

import type { CommentProps } from './comment';
import { Comment } from './comment';

const reader = { id: 'reader', name: 'reader', picture: '', ip: '', admin: false, block: false, verified: false };

function props(change: Partial<CommentProps> = {}): Omit<CommentProps, 'intl'> {
  return {
    view: 'main',
    theme: 'light',
    level: 0,
    isCommentsDisabled: false,
    user: reader,
    data: {
      id: 'c1',
      pid: '',
      text: '<p>text</p>',
      vote: 0,
      score: 2,
      voted_ips: [],
      time: new Date().toISOString(),
      locator: { url: 'http://localhost/post/1', site: 'remark' },
      user: { ...reader, id: 'author', name: 'author' },
    },
    ...change,
  };
}

function Drawn(p: Omit<CommentProps, 'intl'>) {
  return <Comment {...p} intl={useIntl()} />;
}

function PageForm() {
  return (
    <CommentForm
      id="page"
      user={reader}
      theme="light"
      intl={useIntl()}
      onSubmit={() => Promise.resolve()}
      getPreview={() => Promise.resolve('')}
    />
  );
}

function inWidget(node: preact.ComponentChild, store: ReturnType<typeof mockStore>) {
  return (
    <IntlProvider locale="en" messages={en}>
      <Provider store={store}>{node}</Provider>
    </IntlProvider>
  );
}

beforeEach(() => {
  jest.spyOn(emojiData, 'loadEmoji').mockImplementation(() => Promise.resolve({}));
});
afterEach(() => {
  CommentForm.textareaCounter = 0;
  localStorage.clear();
  jest.restoreAllMocks();
});

describe('<Comment /> around the plain comment', () => {
  it('subscribes nothing of its own to the store, even with its reply form open', () => {
    const store = mockStore({ user: reader, theme: 'light' });
    const subscribe = jest.spyOn(store, 'subscribe');

    render(inWidget(<Drawn {...props({ editMode: CommentMode.Reply })} CommentForm={CommentForm} />, store));

    expect(subscribe).not.toHaveBeenCalled();
  });

  it('gives the reply form the reader and the theme of the store as it draws', () => {
    const args = jest.spyOn(email, 'subscribeByEmailArgs');
    const first = { ...reader, email_subscription: false };
    const store = mockStore({ user: first, theme: 'dark' });
    const view = render(
      inWidget(<Drawn {...props({ editMode: CommentMode.Reply })} CommentForm={CommentForm} />, store)
    );
    expect(args).toHaveBeenLastCalledWith(first, 'dark', expect.anything(), expect.any(Function));

    const second = { ...reader, email_subscription: true };
    const next = mockStore({ user: second, theme: 'light' });
    view.rerender(inWidget(<Drawn {...props({ editMode: CommentMode.Reply })} CommentForm={CommentForm} />, next));
    expect(args).toHaveBeenLastCalledWith(second, 'light', expect.anything(), expect.any(Function));
  });

  it('numbers its forms’ fields apart from the other forms of the page', () => {
    const store = mockStore({});
    const view = render(
      inWidget(
        <div>
          <Drawn {...props({ editMode: CommentMode.Reply })} CommentForm={CommentForm} />
          <PageForm />
        </div>,
        store
      )
    );

    // the page's form takes its id as it renders, the comment's as it mounts: what the label and the
    // toolbar need is that no two are the same
    const ids = Array.from(view.container.querySelectorAll('textarea')).map((t) => t.id);
    expect(ids.sort()).toEqual(['textarea_1', 'textarea_2']);
  });

  it('stores the score a vote comes back with', async () => {
    jest.spyOn(api, 'putCommentVote').mockImplementation(() => Promise.resolve({ id: 'c1', score: 3 }));
    const store = mockStore({});
    const view = render(inWidget(<Drawn {...props()} />, store));

    fireEvent.click(view.container.querySelector('[title="Vote up"]')!);
    await act(async () => {});

    expect(store.getActions()).toContainEqual({ type: COMMENT_PATCH, patch: { id: 'c1', score: 3, vote: 1 } });
  });

  it('keeps the plain nodes when the thread draws again', () => {
    const store = mockStore({});
    const view = render(inWidget(<Drawn {...props()} />, store));
    const before = Array.from(view.container.querySelectorAll('*'));

    view.rerender(inWidget(<Drawn {...props({ repliesCount: 2 })} />, store));

    const after = Array.from(view.container.querySelectorAll('*'));
    expect(after).toHaveLength(before.length);
    after.forEach((node, i) => expect(node).toBe(before[i]));
  });
});
