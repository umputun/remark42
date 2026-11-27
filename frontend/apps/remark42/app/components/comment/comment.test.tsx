import { h } from 'preact';
import '@testing-library/jest-dom';
import { act, fireEvent, screen, waitFor } from '@testing-library/preact';

import type { IntlShape } from 'common/intl';
import { createIntl, useIntl } from 'common/intl';
import { render } from 'tests/utils';
import { StaticStore } from 'common/static-store';

import type { CommentProps } from './comment';
import { Comment } from './comment';
import { CommentForm } from 'components/comment-form';
import { CommentMode } from 'common/types';
import * as postMessage from 'utils/post-message';
import de from 'locales/de.json';

function CommentWithIntl(props: CommentProps) {
  const intl = useIntl();

  return <Comment {...props} intl={intl} />;
}

function getProps(): CommentProps {
  return {
    isCommentsDisabled: false,
    theme: 'light',
    post_info: {
      url: 'http://localhost/post/1',
      count: 2,
      read_only: false,
    },
    view: 'main',
    data: {
      id: 'comment_id',
      text: 'test comment',
      vote: 0,
      time: new Date().toString(),
      pid: 'parent_id',
      score: 0,
      voted_ips: [],
      locator: {
        url: 'somelocatorurl',
        site: 'remark',
      },
      user: {
        id: 'someone',
        picture: 'http://localhost/somepicture-url',
        name: 'username',
        ip: '',
        admin: false,
        block: false,
        verified: false,
      },
    },
    user: {
      id: 'testuser',
      picture: 'http://localhost/testuser-url',
      name: 'test',
      ip: '',
      admin: false,
      block: false,
      verified: false,
    },
    intl: {} as IntlShape,
  };
}

describe('<Comment />', () => {
  let props = getProps();

  beforeEach(() => {
    props = getProps();
  });

  it('should render patreon subscriber icon', async () => {
    const props = getProps();
    props.data.user.paid_sub = true;

    render(<CommentWithIntl {...props} />);
    const patreonSubscriberIcon = await screen.findByAltText('Patreon Paid Subscriber');
    expect(patreonSubscriberIcon).toBeVisible();
    expect(patreonSubscriberIcon.tagName).toBe('IMG');
  });

  it('should give the time of the comment in a machine-readable form', () => {
    const props = getProps();
    props.data.time = '2026-11-21T18:05:00+03:00';

    const { container } = render(<CommentWithIntl {...props} />);

    expect(container.querySelector('time')).toHaveAttribute('datetime', '2026-11-21T15:05:00.000Z');
  });

  describe('opening a profile', () => {
    // the message goes to whatever page embeds the widget, and an admin's copy of a comment has
    // the commenter's IP hash in its user record. Whether the author is the reader it does not
    // say, so the two messages differ in the id only
    it.each([
      ['of another user', 'someone'],
      ['of the current user', 'testuser'],
    ])('sends the parent only the profile fields %s', (_label, id) => {
      const postMessageToParent = jest.spyOn(postMessage, 'postMessageToParent').mockReturnValue(true);
      const props = getProps();
      props.user!.admin = true;
      Object.assign(props.data.user, { id, ip: 'ip-hash', verified: true, paid_sub: true, site_id: 'remark' });

      render(<CommentWithIntl {...props} />);
      fireEvent.click(screen.getByText('username'));

      expect(postMessageToParent).toHaveBeenCalledTimes(1);
      expect(postMessageToParent).toHaveBeenCalledWith({
        profile: { id, name: 'username', picture: 'http://localhost/somepicture-url', token: expect.any(String) },
      });
      postMessageToParent.mockRestore();
    });
  });

  describe('voting', () => {
    let props = getProps();

    beforeEach(() => {
      props = getProps();
    });

    it.each([
      [
        'when the comment is pinned',
        () => {
          props.view = 'pinned';
        },
      ],
      [
        'when rendered in profile',
        () => {
          props.view = 'user';
        },
      ],
      [
        'when post is read only',
        () => {
          props.post_info!.read_only = true;
        },
      ],
      [
        'when comment was deleted',
        () => {
          props.data.delete = true;
        },
      ],
      [
        'on current user comments',
        () => {
          props.user!.id = 'testuser';
          props.data.user.id = 'testuser';
        },
      ],
      [
        'for guest users',
        () => {
          props.user = null;
        },
      ],
      [
        'for anonymous users',
        () => {
          props.user!.id = 'anonymous_1';
        },
      ],
    ])('should not render vote component %s', (_, action) => {
      action();
      render(<CommentWithIntl {...props} />);
      expect(screen.queryByTitle('Vote up')).not.toBeInTheDocument();
      expect(screen.queryByTitle('Vote down')).not.toBeInTheDocument();
    });
  });

  it.each([
    [
      'pinned',
      () => {
        props.view = 'pinned';
      },
    ],
    [
      'deleted',
      () => {
        props.data.delete = true;
      },
    ],
    [
      'collapsed',
      () => {
        props.collapsed = true;
      },
    ],
  ])('should not render actions when comment is  %s', (_, mutateProps) => {
    mutateProps();
    render(<CommentWithIntl {...props} />);
    expect(screen.queryByTitle('Reply')).not.toBeInTheDocument();
  });

  it.each([
    ['anonymous', 'anonymous_1', true],
    ['signed in with a provider', 'github_1', false],
  ])('pixelates the avatar of an author only when anonymous: %s', (_, id, pixelated) => {
    props.data.user.id = id;
    const { container } = render(<CommentWithIntl {...props} />);

    const avatar = container.querySelector('img.avatar');
    expect(avatar).toBeInTheDocument();
    expect(avatar?.classList.contains('avatarIdenticon')).toBe(pixelated);
  });

  it('toggles edit mode', async () => {
    props = getProps();
    props.repliesCount = 0;
    props.data.user = props.user!;
    props.data.time = new Date().toString();
    props.setReplyEditState = jest.fn().mockImplementation(() => {
      props.editMode = props.editMode === undefined ? CommentMode.Edit : CommentMode.None;
    });
    StaticStore.config.edit_duration = 300;
    const { rerender } = render(<CommentWithIntl {...props} />);
    fireEvent(screen.getByText('Edit'), new MouseEvent('click', { bubbles: true }));
    rerender(<CommentWithIntl {...props} />);
    await waitFor(() => {
      expect(screen.getByText('Cancel')).toBeVisible();
    });
    fireEvent(screen.getByText('Cancel'), new MouseEvent('click', { bubbles: true }));
    rerender(<CommentWithIntl {...props} />);
    await waitFor(() => {
      expect(screen.getByText('Edit')).toBeVisible();
    });
  });

  describe('edit controls follow the reader while the comment stays mounted', () => {
    // signing in and out reloads the thread in place, so the same component sees the reader change
    function ownRecentComment(): CommentProps {
      const own = getProps();
      own.repliesCount = 0;
      own.data.user = own.user!;
      own.data.time = new Date().toString();
      StaticStore.config.edit_duration = 300;
      return own;
    }

    it('takes them away when the reader signs out', () => {
      const own = ownRecentComment();
      const { rerender } = render(<CommentWithIntl {...own} />);
      expect(screen.getByText('Edit')).toBeVisible();

      rerender(<CommentWithIntl {...own} user={null} />);
      // a later update, such as the reloaded thread arriving, must not bring them back either
      rerender(<CommentWithIntl {...own} user={null} />);

      expect(screen.queryByText('Edit')).not.toBeInTheDocument();
      expect(screen.queryByRole('timer')).not.toBeInTheDocument();
    });

    it('offers them when the author signs in', () => {
      const own = ownRecentComment();
      const { rerender } = render(<CommentWithIntl {...own} user={null} />);
      expect(screen.queryByText('Edit')).not.toBeInTheDocument();

      rerender(<CommentWithIntl {...own} />);

      expect(screen.getByText('Edit')).toBeVisible();
    });
  });

  describe('edit on a read-only post', () => {
    // the author's own comment, still inside the edit window
    function ownRecentComment(readOnly: boolean, admin: boolean): CommentProps {
      const own = getProps();
      own.repliesCount = 0;
      own.user!.admin = admin;
      own.data.user = own.user!;
      own.data.time = new Date().toString();
      own.post_info!.read_only = readOnly;
      StaticStore.config.edit_duration = 300;
      return own;
    }

    it('hides Edit and its countdown from a commenter but keeps Delete', () => {
      render(<CommentWithIntl {...ownRecentComment(true, false)} />);

      expect(screen.queryByText('Edit')).not.toBeInTheDocument();
      expect(screen.queryByRole('timer')).not.toBeInTheDocument();
      expect(screen.getByText('Delete')).toBeVisible();
    });

    it('offers Edit to a commenter on an open post', () => {
      render(<CommentWithIntl {...ownRecentComment(false, false)} />);

      expect(screen.getByText('Edit')).toBeVisible();
      expect(screen.getByRole('timer')).toBeVisible();
    });

    it('offers Edit to an admin', () => {
      render(<CommentWithIntl {...ownRecentComment(true, true)} />);

      expect(screen.getByText('Edit')).toBeVisible();
    });

    it('takes Delete away from a commenter when the edit window ends', () => {
      jest.useFakeTimers();
      try {
        render(<CommentWithIntl {...ownRecentComment(true, false)} />);
        expect(screen.getByText('Delete')).toBeVisible();

        act(() => {
          jest.advanceTimersByTime(301 * 1000);
        });

        expect(screen.queryByText('Delete')).not.toBeInTheDocument();
      } finally {
        jest.useRealTimers();
      }
    });

    it('keeps Delete through an edit window longer than one timeout can wait', () => {
      jest.useFakeTimers();
      try {
        const own = ownRecentComment(true, false);
        StaticStore.config.edit_duration = 30 * 24 * 60 * 60;
        const { unmount } = render(<CommentWithIntl {...own} />);

        act(() => {
          jest.advanceTimersByTime(25 * 24 * 60 * 60 * 1000);
        });

        expect(screen.getByText('Delete')).toBeVisible();
        unmount();
        expect(jest.getTimerCount()).toBe(0);
      } finally {
        StaticStore.config.edit_duration = 300;
        jest.useRealTimers();
      }
    });

    it('takes Delete away at the end of an edit window longer than one timeout can wait', () => {
      jest.useFakeTimers();
      try {
        const own = ownRecentComment(true, false);
        StaticStore.config.edit_duration = 30 * 24 * 60 * 60;
        render(<CommentWithIntl {...own} />);

        act(() => {
          jest.advanceTimersByTime(25 * 24 * 60 * 60 * 1000);
        });
        expect(screen.getByText('Delete')).toBeVisible();

        act(() => {
          jest.advanceTimersByTime(6 * 24 * 60 * 60 * 1000);
        });
        expect(screen.queryByText('Delete')).not.toBeInTheDocument();
      } finally {
        StaticStore.config.edit_duration = 300;
        jest.useRealTimers();
      }
    });

    it('keeps Cancel for a commenter whose editor is open when the post closes', () => {
      const own = ownRecentComment(true, false);
      own.editMode = CommentMode.Edit;
      render(<CommentWithIntl {...own} />);

      expect(screen.getByText('Cancel')).toBeVisible();
    });
  });

  // Regression tests for issue #2040.
  // The edit textarea must reflect `data.orig` byte-for-byte; any transformation
  // (HTML-entity decoding in particular) corrupts user input on save because
  // bluemonday then strips now-real tags that the user typed as entities.
  describe('edit textarea preserves data.orig verbatim', () => {
    afterEach(() => {
      CommentForm.textareaCounter = 0;
      localStorage.clear();
    });

    const cases: Array<[string, string]> = [
      ['issue 2040 canonical', '&lt;script&gt;Hacked you!&lt;/script&gt;'],
      ['doubly-escaped entity', '&amp;lt;script&amp;gt;'],
      ['mixed named entities', 'I love &copy; 2026 &amp; &quot;quotes&quot; &apos;too&apos;'],
      ['nbsp entity whitespace', '&nbsp;&nbsp;&nbsp;indented'],
      ['decimal numeric entities', '&#60;div&#62;hello&#60;/div&#62;'],
      ['hex numeric entities with xss-like content', '&#x3C;img src=x onerror=alert(1)&#x3E;'],
      ['hex numeric emoji entities', '&#x1F600; &#x1F4A9; emoji via numeric'],
      ['obscure named entities', '&AElig;sop &eacute;tait &zwj;ici'],
      ['malformed entity without semicolon', '&lt without semicolon and &amp; with'],
      ['entity-shaped junk', '&;&lt;;&#;&#x;&#xZZZ;'],
      ['recursive-looking numeric entity', '&#38;#60;'],
      ['programmer content with real < and entities', '5 &lt; 10 &amp;&amp; 10 &gt; 5'],
      ['real < mixed with &lt;', 'a < b and &lt;tag&gt; literal'],
      ['sixteen back-to-back &lt;', '&lt;&lt;&lt;&lt;&lt;&lt;&lt;&lt;&lt;&lt;&lt;&lt;&lt;&lt;&lt;&lt;'],
      ['fenced code block with entities', '```\n&lt;pre&gt;code&lt;/pre&gt;\n```'],
      ['link with entity-encoded query ampersands', '[link](https://example.com/?a=1&amp;b=2&amp;c=3)'],
      ['null/replacement/surrogate hex entities', '&#x0;&#xFFFD;&#xD800;'],

      ['zero-width characters interleaved', 'Hello\u200Bworld\u200Cfoo\u200Dbar'],
      ['bidi Hebrew + Arabic + ASCII', '\u05E9\u05DC\u05D5\u05DD hello \u0645\u0631\u062D\u0628\u0627'],
      ['RLO override embedded', 'safe\u202Eevil\u202Cend'],
      ['decomposed vs precomposed é', 'cafe\u0301 vs caf\u00E9'],
      ['ZWJ family emoji', 'family: \uD83D\uDC68\u200D\uD83D\uDC69\u200D\uD83D\uDC66'],
      ['supplementary plane surrogate pairs', 'poop: \uD83D\uDCA9 and math: \uD835\uDC00'],
      ['line and paragraph separators', 'line1\u2028line2\u2029para2'],
      ['BOM at start middle end', '\uFEFFstart mid\uFEFFdle end\uFEFF'],
      ['varied unicode whitespace', 'a\u00A0b\u3000c\u2003d\u202Fe'],
      ['tab LF CR CRLF LFCR', 'tab\there\nnl\rcr\r\ncrlf\n\rlfcr'],
      ['cyrillic homoglyph a', 'Cyrillic \u0430pple vs Latin apple'],
      ['full-width angle brackets', 'fullwidth \uFF1Cscript\uFF1E not a tag'],
      ['bidi mark soup', 'mix: \u202Dltr\u202C \u202Ertl\u202C \u200E\u200F'],

      ['inline code with real script tag', '`<script>alert(1)</script>`'],
      ['inline code with entity script tag', '`&lt;script&gt;`'],
      [
        'fenced html with iframe and double-escaped',
        '```html\n<iframe src="javascript:alert(1)"></iframe>\n&amp;lt;b&amp;gt;\n```',
      ],
      ['javascript link', '[click](javascript:alert(1))'],
      ['image with entity alt and title', '![&lt;alt&gt;](x.png "&amp;title&amp;")'],
      ['markdown backslash escapes', '\\*not em\\* \\_not em\\_ \\\\ \\< \\& \\`not code\\`'],
      ['entities inside emphasis markers', '*&lt;em&gt;* _&gt;underscore&lt;_ **&amp;bold&amp;**'],
      ['headers with entities', '# &lt;h1&gt;\n## &amp;header&amp;'],
      ['blockquote with entities multiline', '> &lt;foo&gt;\n> &amp;quoted&amp;\n>\n> nested &lt;bar/&gt;'],
      ['raw autolinks', '<https://example.com/?a=1&b=2> see also <user@example.com>'],
      ['raw script tag no markdown', '<script>alert(1)</script>'],
      ['iframe object embed chain', '<iframe src=x></iframe><object data=x></object><embed src=x>'],
      ['kitchen sink', 'mix: `a` \\* *b* &lt;c&gt; &amp;d&amp; \\\\ \\`e\\` [f](javascript:0) ![g](h "&quot;")'],

      ['empty string', ''],
      ['only whitespace', '   \t\n   '],
    ];

    // HTML5 textarea.value always normalises \r\n and lone \r to \n.
    // This happens inside the browser regardless of any remark42 code,
    // so the edit round-trip guarantee is "byte-equal after newline normalisation".
    const expectedTextareaValue = (raw: string) => raw.replace(/\r\n|\r/g, '\n');

    it.each(cases)('renders unchanged: %s', (_label, payload) => {
      CommentForm.textareaCounter = 0;
      StaticStore.config.edit_duration = 300;

      const p = getProps();
      p.repliesCount = 0;
      p.user!.id = '100';
      p.data.user.id = '100';
      p.editMode = CommentMode.Edit;
      p.CommentForm = CommentForm;
      Object.assign(p.data, {
        id: '101',
        vote: 1,
        time: Date.now(),
        delete: false,
        orig: payload,
      });

      render(<CommentWithIntl {...p} />);

      const textarea = screen.getByTestId('textarea_1') as HTMLTextAreaElement;
      expect(textarea.value).toBe(expectedTextareaValue(payload));
    });
  });

  describe('block confirmation', () => {
    // the duration is a select label reused inside the sentence: only its first letter may be
    // lowered, or a language that capitalises nouns, such as German, loses those capitals
    it.each([
      ['en', {}, 'Do you want to block username for a month?'],
      ['de', de, 'Möchten Sie username wirklich für einen Monat sperren?'],
    ])('inserts the duration label with only its first letter lowered (%s)', async (locale, messages, expected) => {
      const confirm = jest.spyOn(window, 'confirm').mockReturnValue(false);
      const props = getProps();
      props.user!.admin = true;
      const { container } = render(<Comment {...props} intl={createIntl(locale, messages)} />);

      fireEvent.change(container.querySelector('select')!, { target: { value: '43200m' } });

      await waitFor(() => expect(confirm).toHaveBeenCalledWith(expected));
      confirm.mockRestore();
    });
  });
});
