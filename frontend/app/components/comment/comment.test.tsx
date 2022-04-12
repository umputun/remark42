import { h } from 'preact';
import { mount } from 'enzyme';
import '@testing-library/jest-dom';
import { screen } from '@testing-library/preact';
import { render } from 'tests/utils';

import { useIntl, IntlProvider, IntlShape } from 'react-intl';
import { Provider } from 'react-redux';

import enMessages from 'locales/en.json';
import { StaticStore } from 'common/static-store';

import { Comment, CommentProps } from './comment';
import { mockStore } from '__stubs__/store';

function CommentWithIntl(props: CommentProps) {
  return <Comment {...props} intl={useIntl()} />;
}

// @depricated
function mountComment(props: CommentProps) {
  return mount(
    <IntlProvider locale="en" messages={enMessages}>
      <Provider store={mockStore({})}>
        <CommentWithIntl {...props} />
      </Provider>
    </IntlProvider>
  );
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
  describe('admin controls', () => {
    it('for admin if shows admin controls', () => {
      props.user!.admin = true;
      const element = mountComment(props);
      const controls = element.find('.comment__controls').children();
      expect(controls.length).toBe(5);
      expect(controls.at(0).text()).toEqual('Copy');
      expect(controls.at(1).text()).toEqual('Pin');
      expect(controls.at(2).text()).toEqual('Hide');
      expect(controls.at(3).getDOMNode().childNodes[0].textContent).toEqual('Block');
      expect(controls.at(4).text()).toEqual('Delete');
    });

    it('for regular user it shows only "hide"', () => {
      const element = mountComment(props);

      const controls = element.find('.comment__controls').children();
      expect(controls.length).toBe(1);
      expect(controls.at(0).text()).toEqual('Hide');
    });

    it('should be editable', async () => {
      StaticStore.config.edit_duration = 300;

      props.repliesCount = 0;
      props.user!.id = '100';
      props.data.user.id = '100';
      Object.assign(props.data, {
        id: '101',
        vote: 1,
        time: new Date().toString(),
        delete: false,
        orig: 'test',
      });

      render(<CommentWithIntl {...props} />);
      // it can be less than 300 due to test checks time
      expect(['299', '300']).toContain(screen.getByRole('timer').innerText);
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

      const component = mountComment(props);
      expect(component.find('Comment').state('editDeadline')).toBe(null);
    });
  });
});
