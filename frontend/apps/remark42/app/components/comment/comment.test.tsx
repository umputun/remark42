import { h } from 'preact';
import '@testing-library/jest-dom';
import { fireEvent, screen, waitFor } from '@testing-library/preact';
import { useIntl, IntlShape } from 'react-intl';

import { render } from 'tests/utils';
import { StaticStore } from 'common/static-store';

import { Comment, CommentProps } from './comment';
import { CommentForm } from 'components/comment-form';
import { CommentMode } from 'common/types';

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

  describe('verification', () => {
    it('should render active verification icon', () => {
      props.data.user.verified = true;
      render(<CommentWithIntl {...props} />);
      expect(screen.getByTitle('Verified user')).toBeVisible();
    });

    it('should not render verification icon', () => {
      const props = getProps();
      render(<CommentWithIntl {...props} />);
      expect(screen.queryByTitle('Verified user')).not.toBeInTheDocument();
    });

    it('should render verification button for admin', () => {
      props.user!.admin = true;
      render(<CommentWithIntl {...props} />);
      expect(screen.getByTitle('Toggle verification')).toBeVisible();
    });

    it('should render active verification icon for admin', () => {
      props.user!.admin = true;
      props.data.user.verified = true;
      render(<CommentWithIntl {...props} />);
      expect(screen.queryByTitle('Verified user')).toBeVisible();
    });
  });

  describe('voting', () => {
    let props = getProps();

    beforeEach(() => {
      props = getProps();
    });

    it('should render vote component', () => {
      render(<CommentWithIntl {...props} />);
      expect(screen.getByTitle('Votes score')).toBeVisible();
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
        'when rendered in preview',
        () => {
          props.view = 'preview';
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
      expect(screen.queryByText('Votes score')).not.toBeInTheDocument();
    });
  });

  it('should render action buttons', () => {
    render(<CommentWithIntl {...props} />);
    expect(screen.getByText('Reply')).toBeVisible();
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

  it('should be editable', async () => {
    StaticStore.config.edit_duration = 300;

    props.repliesCount = 0;
    props.user!.id = '100';
    props.data.user.id = '100';
    Object.assign(props.data, {
      id: '101',
      vote: 1,
      time: Date.now(),
      delete: false,
      orig: 'test',
    });

    render(<CommentWithIntl {...props} />);
    expect(screen.getByText('Edit')).toBeVisible();
  });

  it('should not be editable', () => {
    StaticStore.config.edit_duration = 300;
    Object.assign(props.data, {
      user: props.user,
      id: '100',
      vote: 1,
      time: new Date(new Date().getDate() - 300).toString(),
      orig: 'test',
    });

    render(<CommentWithIntl {...props} />);
    expect(screen.queryByRole('timer')).not.toBeInTheDocument();
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
      // @ts-ignore - CommentForm prop is optional on CommentProps
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
});
