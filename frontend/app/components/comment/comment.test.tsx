import { h } from 'preact';
import { mount } from 'enzyme';
import { useIntl, IntlProvider } from 'react-intl';

import enMessages from 'locales/en.json';
import type { User, Comment as CommentType, PostInfo } from 'common/types';
import { StaticStore } from 'common/static-store';
import { sleep } from 'utils/sleep';

import { Comment, CommentProps } from './comment';

function mountComment(props: CommentProps) {
  function Wrapper(updateProps: Partial<CommentProps> = {}) {
    const intl = useIntl();

    return (
      <IntlProvider locale="en" messages={enMessages}>
        <Comment {...props} {...updateProps} intl={intl} />
      </IntlProvider>
    );
  }

  return mount(<Wrapper />);
}

const DefaultProps: Partial<CommentProps> = {
  CommentForm: null,
  post_info: {
    read_only: false,
  } as PostInfo,
  view: 'main',
  data: {
    text: 'test comment',
    vote: 0,
    user: {
      id: 'someone',
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
} as CommentProps;

describe('<Comment />', () => {
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

    it('verification badge clickable for admin', () => {
      const element = mountComment({ ...DefaultProps, user: { ...DefaultProps.user, admin: true } } as CommentProps);

      const controls = element.find('.comment__verification').first();
      expect(controls.hasClass('comment__verification_clickable')).toEqual(true);
    });

    it('verification badge not clickable for regular user', () => {
      const element = mountComment({
        ...DefaultProps,
        data: { ...DefaultProps.data, user: { ...DefaultProps.data!.user, verified: true } },
      } as CommentProps);

      const controls = element.find('.comment__verification').first();
      expect(controls.hasClass('comment__verification_clickable')).toEqual(false);
    });

    it('should be editable', () => {
      StaticStore.config.edit_duration = 300;

      const initTime = new Date().toString();
      const changedTime = new Date(Date.now() + 10 * 1000).toString();
      const props = {
        ...DefaultProps,
        user: DefaultProps.user as User,
        data: {
          ...DefaultProps.data,
          id: '100',
          user: DefaultProps.user as User,
          vote: 1,
          time: initTime,
          delete: false,
          orig: 'test',
        } as CommentType,
        repliesCount: 0,
      } as CommentProps;
      const component = mountComment(props);
      const comment = component.find(Comment);

      expect((comment.state('editDeadline') as Date).getTime()).toBe(
        new Date(new Date(initTime).getTime() + 300 * 1000).getTime()
      );

      component.setProps({ data: { ...props.data, time: changedTime } });

      expect((comment.state('editDeadline') as Date).getTime()).toBe(
        new Date(new Date(changedTime).getTime() + 300 * 1000).getTime()
      );
    });

    it('shoud not be editable', () => {
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
