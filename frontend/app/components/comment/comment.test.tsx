import { h } from 'preact';
import { mount } from 'enzyme';
import '@testing-library/jest-dom';
import { screen } from '@testing-library/preact';
import { render } from 'tests/utils';

import { useIntl, IntlProvider } from 'react-intl';

import enMessages from 'locales/en.json';
import type { User, Comment as CommentType, PostInfo } from 'common/types';
import { StaticStore } from 'common/static-store';
import { sleep } from 'utils/sleep';

import { Comment, CommentProps } from './comment';

function CommentWithIntl(props: CommentProps) {
  return <Comment {...props} intl={useIntl()} />;
}

// @depricated
function mountComment(props: CommentProps) {
  return mount(
    <IntlProvider locale="en" messages={enMessages}>
      <CommentWithIntl {...props} />
    </IntlProvider>
  );
}

function getDefaultProps() {
  return {
    post_info: {
      read_only: false,
    } as PostInfo,
    view: 'main',
    data: {
      text: 'test comment',
      vote: 0,
      user: {
        id: 'someone',
        name: 'username',
        picture: 'somepicture-url',
      },
      time: new Date().toString(),
      locator: {
        url: 'somelocatorurl',
        site: 'remark',
      },
    } as CommentType,
    user: {
      admin: false,
      id: 'testuser',
      picture: 'somepicture-url',
    } as User,
  } as CommentProps & { user: User };
}

const DefaultProps = getDefaultProps();
describe('<Comment />', () => {
  it('should render patreon subscriber icon', async () => {
    const props = getDefaultProps() as CommentProps;
    props.data.user.paid_sub = true;

    render(<CommentWithIntl {...props} />);
    const patreonSubscriberIcon = await screen.findByAltText('Patreon Paid Subscriber');
    expect(patreonSubscriberIcon).toBeVisible();
    expect(patreonSubscriberIcon.tagName).toBe('IMG');
  });

  describe('verification', () => {
    it('should render active verification icon', () => {
      const props = getDefaultProps();
      props.data.user.verified = true;
      render(<CommentWithIntl {...props} />);
      expect(screen.getByTitle('Verified user')).toBeVisible();
    });

    it('should not render verification icon', () => {
      const props = getDefaultProps();
      render(<CommentWithIntl {...props} />);
      expect(screen.queryByTitle('Verified user')).not.toBeInTheDocument();
    });

    it('should render verification button for admin', () => {
      const props = getDefaultProps();
      props.user.admin = true;
      render(<CommentWithIntl {...props} />);
      expect(screen.getByTitle('Toggle verification')).toBeVisible();
    });

    it('should render active verification icon for admin', () => {
      const props = getDefaultProps();
      props.user.admin = true;
      props.data.user.verified = true;
      render(<CommentWithIntl {...props} />);
      expect(screen.queryByTitle('Verified user')).toBeVisible();
    });
  });

  describe('voting', () => {
    it('should be disabled for an anonymous user', () => {
      const wrapper = mountComment({ ...DefaultProps, user: { id: 'anonymous_1' } } as CommentProps);
      const voteButtons = wrapper.find('.comment__vote');

      expect(voteButtons.length).toEqual(2);

      voteButtons.forEach((button) => {
        expect(button.prop('aria-disabled')).toEqual('true');
        expect(button.prop('title')).toEqual("Anonymous users can't vote");
      });
    });

    it('should be enabled for an anonymous user when it was allowed from server', () => {
      StaticStore.config.anon_vote = true;

      const wrapper = mountComment({ ...DefaultProps, user: { id: 'anonymous_1' } } as CommentProps);
      const voteButtons = wrapper.find('.comment__vote');

      expect(voteButtons.length).toEqual(2);

      voteButtons.forEach((button) => {
        expect(button.prop('aria-disabled')).toEqual('false');
      });
    });

    it('disabled on user info widget', () => {
      const element = mountComment({ ...DefaultProps, view: 'user' } as CommentProps);

      const voteButtons = element.find('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      voteButtons.forEach((b) => {
        expect(b.getDOMNode().getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getDOMNode().getAttribute('title')).toStrictEqual("Voting allowed only on post's page");
      });
    });

    it('disabled on read only post', () => {
      const element = mountComment({
        ...DefaultProps,
        post_info: { ...DefaultProps.post_info, read_only: true },
      } as CommentProps);

      const voteButtons = element.find('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      voteButtons.forEach((b) => {
        expect(b.getDOMNode().getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getDOMNode().getAttribute('title')).toStrictEqual("Can't vote on read-only topics");
      });
    });

    it('disabled for deleted comment', () => {
      const element = mountComment({ ...DefaultProps, data: { ...DefaultProps.data, delete: true } } as CommentProps);

      const voteButtons = element.find('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      voteButtons.forEach((b) => {
        expect(b.getDOMNode().getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getDOMNode().getAttribute('title')).toStrictEqual("Can't vote for deleted comment");
      });
    });

    it('disabled for guest', () => {
      const element = mountComment({
        ...DefaultProps,
        user: {
          id: 'someone',
          picture: 'somepicture-url',
        },
      } as CommentProps);

      const voteButtons = element.find('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      voteButtons.forEach((b) => {
        expect(b.getDOMNode().getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getDOMNode().getAttribute('title')).toStrictEqual("Can't vote for your own comment");
      });
    });

    it('disabled for own comment', () => {
      const element = mountComment({ ...DefaultProps, user: null } as CommentProps);

      const voteButtons = element.find('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      voteButtons.forEach((b) => {
        expect(b.getDOMNode().getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getDOMNode().getAttribute('title')).toStrictEqual('Sign in to vote');
      });
    });

    it('disabled for already upvoted comment', async () => {
      const voteSpy = jest.fn(async () => undefined);
      const element = mountComment({
        ...DefaultProps,
        data: { ...DefaultProps.data, vote: +1 } as CommentProps['data'],
        putCommentVote: voteSpy,
      } as CommentProps);

      const voteButtons = element.find('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      expect(voteButtons.at(0).getDOMNode().getAttribute('aria-disabled')).toStrictEqual('true');
      voteButtons.at(0).simulate('click');
      await sleep(100);
      expect(voteSpy).not.toBeCalled();

      expect(voteButtons.at(1).getDOMNode().getAttribute('aria-disabled')).toStrictEqual('false');
      voteButtons.at(1).simulate('click');
      await sleep(100);
      expect(voteSpy).toBeCalled();
    }, 30000);

    it('disabled for already downvoted comment', async () => {
      const voteSpy = jest.fn(async () => undefined);
      const element = mountComment({
        ...DefaultProps,
        data: {
          ...DefaultProps.data,
          vote: -1,
        },
        putCommentVote: voteSpy,
      } as CommentProps);

      const voteButtons = element.find('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      expect(voteButtons.at(1).getDOMNode().getAttribute('aria-disabled')).toStrictEqual('true');
      voteButtons.at(1).simulate('click');
      await sleep(100);
      expect(voteSpy).not.toBeCalled();

      expect(voteButtons.at(0).getDOMNode().getAttribute('aria-disabled')).toStrictEqual('false');
      voteButtons.at(0).simulate('click');
      await sleep(100);
      expect(voteSpy).toBeCalled();
    }, 30000);
  });

  describe('admin controls', () => {
    it('for admin if shows admin controls', () => {
      const element = mountComment({ ...DefaultProps, user: { ...DefaultProps.user, admin: true } } as CommentProps);

      const controls = element.find('.comment__controls').children();

      expect(controls.length).toBe(5);
      expect(controls.at(0).text()).toEqual('Copy');
      expect(controls.at(1).text()).toEqual('Pin');
      expect(controls.at(2).text()).toEqual('Hide');
      expect(controls.at(3).getDOMNode().childNodes[0].textContent).toEqual('Block');
      expect(controls.at(4).text()).toEqual('Delete');
    });

    it('for regular user it shows only "hide"', () => {
      const element = mountComment({ ...DefaultProps, user: { ...DefaultProps.user, admin: false } } as CommentProps);

      const controls = element.find('.comment__controls').children();
      expect(controls.length).toBe(1);
      expect(controls.at(0).text()).toEqual('Hide');
    });

    it('should be editable', async () => {
      StaticStore.config.edit_duration = 300;

      const props = getDefaultProps();
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

      const component = mountComment({
        ...DefaultProps,
        user: DefaultProps.user as User,
        data: {
          ...DefaultProps.data,
          id: '100',
          user: DefaultProps.user as User,
          vote: 1,
          time: new Date(new Date().getDate() - 300).toString(),
          orig: 'test',
        } as CommentType,
      } as CommentProps);

      expect(component.find('Comment').state('editDeadline')).toBe(null);
    });
  });
});
