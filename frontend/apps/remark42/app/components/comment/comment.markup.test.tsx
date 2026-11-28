import { h } from 'preact';
import { act, fireEvent } from '@testing-library/preact';

import { useIntl } from 'common/intl';
import { StaticStore } from 'common/static-store';
import { CommentMode } from 'common/types';
import * as copy from 'common/copy';
import { render } from 'tests/utils';
import { CommentForm } from 'components/comment-form';
import * as emojiData from 'components/comment-form/emoji-data';

import type { CommentProps } from './comment';
import { Comment } from './comment';

// the moment the tests run at, so the edit window and its countdown come out the same every time
const NOW = Date.parse('2026-11-21T15:05:00.000Z');

function Drawn(props: CommentProps) {
  return <Comment {...props} intl={useIntl()} />;
}

function props(): CommentProps {
  return {
    isCommentsDisabled: false,
    theme: 'light',
    post_info: { url: 'http://localhost/post/1', count: 2, read_only: false },
    view: 'main',
    level: 0,
    repliesCount: 0,
    data: {
      id: 'comment_id',
      pid: '',
      text: '<p>test comment</p>',
      orig: 'test comment',
      vote: 0,
      time: new Date(NOW - 60 * 1000).toISOString(),
      score: 2,
      voted_ips: [],
      locator: { url: 'http://localhost/post/1', site: 'remark' },
      user: {
        id: 'someone',
        picture: 'http://localhost/someone.png',
        name: 'username',
        ip: '',
        admin: false,
        block: false,
        verified: false,
      },
    },
    user: {
      id: 'reader',
      picture: 'http://localhost/reader.png',
      name: 'reader',
      ip: '',
      admin: false,
      block: false,
      verified: false,
    },
    intl: undefined as never,
  };
}

/** the reader is the author */
function own(p: CommentProps) {
  p.data.user = { ...p.user! };
  return p;
}

/**
 * The comment's HTML. The date and time it shows follow the time zone the tests run in, and the
 * suggestion list of an open form takes a random id; neither is part of what the comment draws.
 */
function markup(container: Element) {
  return container.innerHTML
    .replace(/(<time[^>]*>)[^<]*(<\/time>)/g, '$1TIME$2')
    .replace(/text-expander-\d+/g, 'text-expander-ID');
}

beforeEach(() => {
  jest.spyOn(Date, 'now').mockReturnValue(NOW);
  jest.spyOn(emojiData, 'loadEmoji').mockImplementation(() => Promise.resolve({}));
});
afterEach(() => {
  CommentForm.textareaCounter = 0;
  localStorage.clear();
  jest.restoreAllMocks();
});

type Change = (p: CommentProps) => void;

// the markup the preact component drew, recorded before it was replaced; innerHTML keeps the
// attributes in the order they were written
describe('the comment draws the markup of the preact component it replaces', () => {
  it.each<[string, Change]>([
    ['a reader on another reader’s comment', () => {}],
    [
      'signed out',
      (p) => {
        p.user = null;
      },
    ],
    [
      'an anonymous reader',
      (p) => {
        p.user!.id = 'anonymous_1';
      },
    ],
    [
      'an anonymous reader where anonymous votes are allowed',
      (p) => {
        p.user!.id = 'anonymous_1';
        StaticStore.config.anon_vote = true;
      },
    ],
    [
      'an admin on an unverified author',
      (p) => {
        p.user!.admin = true;
      },
    ],
    [
      'an admin on a verified author',
      (p) => {
        p.user!.admin = true;
        p.data.user.verified = true;
      },
    ],
    [
      'a reader on a verified author',
      (p) => {
        p.data.user.verified = true;
      },
    ],
    [
      'an author who is an admin',
      (p) => {
        p.data.user.admin = true;
      },
    ],
    [
      'a Patreon subscriber',
      (p) => {
        p.data.user.paid_sub = true;
      },
    ],
    [
      'a picture-less author',
      (p) => {
        p.data.user.picture = '';
      },
    ],
    [
      'an anonymous author',
      (p) => {
        p.data.user.id = 'anonymous_2';
      },
    ],
    ['own comment in its edit window', own],
    [
      'own comment with replies',
      (p) => {
        own(p);
        p.repliesCount = 1;
      },
    ],
    [
      'own comment after its edit window',
      (p) => {
        own(p);
        p.data.time = new Date(NOW - 600 * 1000).toISOString();
      },
    ],
    [
      'own comment of an admin who may always edit',
      (p) => {
        own(p);
        p.user!.admin = true;
        p.data.user.admin = true;
        p.repliesCount = 3;
        StaticStore.config.admin_edit = true;
      },
    ],
    [
      'own comment on a read-only post',
      (p) => {
        own(p);
        p.post_info!.read_only = true;
      },
    ],
    [
      'a read-only post',
      (p) => {
        p.post_info!.read_only = true;
      },
    ],
    [
      'an admin on a read-only post',
      (p) => {
        p.user!.admin = true;
        p.post_info!.read_only = true;
      },
    ],
    [
      'pinned',
      (p) => {
        p.view = 'pinned';
        p.disabled = true;
      },
    ],
    [
      'pinned and collapsed',
      (p) => {
        p.view = 'pinned';
        p.collapsed = true;
      },
    ],
    [
      'in a profile, with the page title',
      (p) => {
        p.view = 'user';
        p.user = null;
        p.data.title = 'The post';
      },
    ],
    [
      'in a profile, without a title',
      (p) => {
        p.view = 'user';
        p.user = null;
      },
    ],
    [
      'deleted, to a reader',
      (p) => {
        p.data.delete = true;
        p.data.text = '';
      },
    ],
    [
      'deleted, to an admin',
      (p) => {
        p.user!.admin = true;
        p.data.delete = true;
        p.data.text = '';
      },
    ],
    [
      'a blocked author, to a reader',
      (p) => {
        p.isUserBanned = true;
      },
    ],
    [
      'a blocked author, to an admin',
      (p) => {
        p.user!.admin = true;
        p.isUserBanned = true;
      },
    ],
    [
      'a blocked author in a profile',
      (p) => {
        p.view = 'user';
        p.isUserBanned = true;
      },
    ],
    [
      'a low score',
      (p) => {
        p.data.score = -10;
      },
    ],
    [
      'a low score, pinned',
      (p) => {
        p.data.score = -10;
        p.data.pin = true;
      },
    ],
    [
      'a pinned comment, to an admin',
      (p) => {
        p.user!.admin = true;
        p.data.pin = true;
      },
    ],
    [
      'disabled',
      (p) => {
        p.disabled = true;
        p.data.score = -10;
      },
    ],
    [
      'collapsed',
      (p) => {
        p.collapsed = true;
      },
    ],
    [
      'collapsed and deleted',
      (p) => {
        p.collapsed = true;
        p.data.delete = true;
        p.data.text = '';
      },
    ],
    [
      'a reply',
      (p) => {
        p.level = 1;
        p.data.pid = 'parent_id';
      },
    ],
    [
      'a reply in a profile',
      (p) => {
        p.level = 1;
        p.view = 'user';
        p.data.pid = 'parent_id';
      },
    ],
    [
      'the deepest level',
      (p) => {
        p.level = 6;
        p.data.pid = 'parent_id';
      },
    ],
    [
      'the dark theme',
      (p) => {
        p.theme = 'dark';
      },
    ],
    [
      'a class from the caller',
      (p) => {
        p.mix = 'extra';
      },
    ],
    [
      'voted up',
      (p) => {
        p.data.vote = 1;
        p.data.score = 3;
      },
    ],
    [
      'voted down, with controversy',
      (p) => {
        p.data.vote = -1;
        p.data.score = -1;
        p.data.controversy = 1.5;
      },
    ],
    [
      'a text with an image, code and a link',
      (p) => {
        p.data.text =
          '<p>Look <a href="https://example.com/">here</a></p><pre><code class="language-go">x := 1</code></pre>' +
          '<p><img src="http://localhost/api/v1/img?src=aHR0cA" alt="a picture" width="20" height="10"></p>';
      },
    ],
    [
      'a time that cannot be read',
      (p) => {
        p.data.time = 'not a time';
      },
    ],
  ])('%s', (_, change) => {
    const p = props();
    change(p);
    expect(markup(render(<Drawn {...p} />).container)).toMatchSnapshot();
  });

  describe('with a form', () => {
    it.each<[string, Change]>([
      [
        'replying',
        (p) => {
          p.editMode = CommentMode.Reply;
        },
      ],
      [
        'replying as an anonymous reader',
        (p) => {
          p.editMode = CommentMode.Reply;
          p.user!.id = 'anonymous_1';
        },
      ],
      [
        'replying in the dark theme',
        (p) => {
          p.editMode = CommentMode.Reply;
          p.theme = 'dark';
        },
      ],
      [
        'editing own comment',
        (p) => {
          own(p);
          p.editMode = CommentMode.Edit;
        },
      ],
      [
        'editing own reply',
        (p) => {
          own(p);
          p.data.pid = 'parent_id';
          p.level = 1;
          p.editMode = CommentMode.Edit;
        },
      ],
      [
        'editing own comment after its window',
        (p) => {
          own(p);
          p.data.time = new Date(NOW - 600 * 1000).toISOString();
          p.editMode = CommentMode.Edit;
        },
      ],
      [
        'editing on a post that became read-only',
        (p) => {
          own(p);
          p.post_info!.read_only = true;
          p.editMode = CommentMode.Edit;
        },
      ],
      [
        'replying in a pinned comment',
        (p) => {
          p.view = 'pinned';
          p.editMode = CommentMode.Reply;
        },
      ],
      [
        'replying while out of view',
        (p) => {
          p.inView = false;
          p.editMode = CommentMode.Reply;
        },
      ],
    ])('%s', (_, change) => {
      const p = props();
      change(p);
      expect(markup(render(<Drawn {...p} CommentForm={CommentForm} />).container)).toMatchSnapshot();
    });

    it('replying without a form to draw', () => {
      const p = props();
      p.editMode = CommentMode.Reply;
      expect(markup(render(<Drawn {...p} />).container)).toMatchSnapshot();
    });
  });

  describe('as it changes', () => {
    it('out of view, then out of view again, then in view', () => {
      const p = props();
      const { container, rerender } = render(<Drawn {...p} inView={false} />);
      expect(markup(container)).toMatchSnapshot('out of view');

      rerender(<Drawn {...p} inView={false} data={{ ...p.data, score: 3 }} />);
      expect(markup(container)).toMatchSnapshot('out of view, drawn again');

      rerender(<Drawn {...p} inView={true} />);
      expect(markup(container)).toMatchSnapshot('in view');

      rerender(<Drawn {...p} inView={false} />);
      expect(markup(container)).toMatchSnapshot('out of view after it was in view');
    });

    it('from reading to replying to editing and back', () => {
      const p = own(props());
      const { container, rerender } = render(<Drawn {...p} CommentForm={CommentForm} />);

      rerender(<Drawn {...p} CommentForm={CommentForm} editMode={CommentMode.Reply} />);
      expect(markup(container)).toMatchSnapshot('replying');

      rerender(<Drawn {...p} CommentForm={CommentForm} editMode={CommentMode.Edit} />);
      expect(markup(container)).toMatchSnapshot('editing');

      rerender(<Drawn {...p} CommentForm={CommentForm} editMode={CommentMode.None} />);
      expect(markup(container)).toMatchSnapshot('reading again');
    });

    it('collapsing and expanding, and a theme change', () => {
      const p = props();
      const { container, rerender } = render(<Drawn {...p} />);

      rerender(<Drawn {...p} collapsed={true} />);
      expect(markup(container)).toMatchSnapshot('collapsed');

      rerender(<Drawn {...p} collapsed={false} theme="dark" />);
      expect(markup(container)).toMatchSnapshot('expanded, dark');
    });

    it('an admin toggling the author’s verification and blocking them', () => {
      const p = props();
      p.user!.admin = true;
      const { container, rerender } = render(<Drawn {...p} />);

      rerender(<Drawn {...p} data={{ ...p.data, user: { ...p.data.user, verified: true } }} />);
      expect(markup(container)).toMatchSnapshot('verified');

      rerender(<Drawn {...p} isUserBanned={true} />);
      expect(markup(container)).toMatchSnapshot('blocked');

      rerender(<Drawn {...p} data={{ ...p.data, delete: true, text: '' }} />);
      expect(markup(container)).toMatchSnapshot('deleted');
    });

    it('a new text', () => {
      const p = props();
      const { container, rerender } = render(<Drawn {...p} />);

      rerender(<Drawn {...p} data={{ ...p.data, text: '<p>edited <em>text</em></p>' }} />);
      expect(markup(container)).toMatchSnapshot();
    });

    it('the edit window ending', () => {
      jest.useFakeTimers({ now: NOW });
      try {
        const p = own(props());
        p.data.time = new Date(NOW - 298 * 1000).toISOString();
        const { container } = render(<Drawn {...p} />);
        expect(markup(container)).toMatchSnapshot('two seconds left');

        act(() => {
          jest.advanceTimersByTime(3000);
        });
        expect(markup(container)).toMatchSnapshot('ended');
      } finally {
        jest.useRealTimers();
      }
    });

    it('copied by an admin', async () => {
      jest.spyOn(copy, 'copy').mockImplementation(() => Promise.resolve());
      const p = props();
      p.user!.admin = true;
      const { container, getByText } = render(<Drawn {...p} />);

      await act(async () => {
        fireEvent.click(getByText('Copy'));
      });
      expect(markup(container)).toMatchSnapshot();
    });
  });
});
