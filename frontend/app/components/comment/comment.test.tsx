import { h } from 'preact';
import '@testing-library/jest-dom';
import { screen } from '@testing-library/preact';
import { useIntl, IntlShape } from 'react-intl';

import { render } from 'tests/utils';
import { StaticStore } from 'common/static-store';

import { Comment, CommentProps } from './comment';

function CommentWithIntl(props: CommentProps) {
  return <Comment {...props} intl={useIntl()} />;
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
});
