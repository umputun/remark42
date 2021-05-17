import { h } from 'preact';
import { fireEvent, waitFor } from '@testing-library/preact';

import type { User, Comment as CommentType, PostInfo } from 'common/types';
import { StaticStore } from 'common/static-store';

import { Comment, CommentProps } from './comment';
import { render } from 'tests/utils';

const DefaultProps: Partial<CommentProps> = {
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
      const props = { ...DefaultProps, user: { id: 'anonymous_1' } } as CommentProps;
      const { container } = render(<Comment {...props} />);
      const voteButtons = container.querySelectorAll('.comment__vote');

      expect(voteButtons).toHaveLength(2);

      Array.from(voteButtons).forEach((b) => {
        console.log(b);
        expect(b).toHaveAttribute('aria-disabled', 'true');
        expect(b).toHaveAttribute('title', "Anonymous users can't vote");
      });
    });

    it('should be enabled for an anonymous user when it was allowed from server', () => {
      StaticStore.config.anon_vote = true;

      const props = { ...DefaultProps, user: { id: 'anonymous_1' } } as CommentProps;
      const { container } = render(<Comment {...props} />);
      const voteButtons = container.querySelectorAll('.comment__vote');

      expect(voteButtons).toHaveLength(2);

      voteButtons.forEach((button) => {
        expect(button).toHaveAttribute('aria-disabled', 'false');
      });
    });

    it('disabled on user info widget', () => {
      const { container } = render(<Comment {...({ ...DefaultProps, view: 'user' } as CommentProps)} />);

      const voteButtons = container.querySelectorAll('.comment__vote');

      expect(voteButtons).toHaveLength(2);
      voteButtons.forEach((b) => {
        expect(b).toHaveAttribute('aria-disabled', 'true');
        expect(b).toHaveAttribute('title', "Voting allowed only on post's page");
      });
    });

    it('disabled on read only post', () => {
      const { container } = render(
        <Comment
          {...({ ...DefaultProps, post_info: { ...DefaultProps.post_info, read_only: true } } as CommentProps)}
        />
      );

      const voteButtons = container.querySelectorAll('.comment__vote');

      expect(voteButtons).toHaveLength(2);
      voteButtons.forEach((b) => {
        expect(b).toHaveAttribute('aria-disabled', 'true');
        expect(b).toHaveAttribute('title', "Can't vote on read-only topics");
      });
    });

    it('disabled for deleted comment', () => {
      const { container } = render(
        // ahem
        <Comment {...({ ...DefaultProps, data: { ...DefaultProps.data, delete: true } } as CommentProps)} />
      );

      const voteButtons = container.querySelectorAll('.comment__vote');

      expect(voteButtons).toHaveLength(2);
      voteButtons.forEach((b) => {
        expect(b).toHaveAttribute('aria-disabled', 'true');
        expect(b).toHaveAttribute('title', "Can't vote for deleted comment");
      });
    });

    it('disabled for guest', () => {
      const { container } = render(
        <Comment
          {...({
            ...DefaultProps,
            user: {
              id: 'someone',
              picture: 'somepicture-url',
            },
          } as CommentProps)}
        />
      );

      const voteButtons = container.querySelectorAll('.comment__vote');

      expect(voteButtons).toHaveLength(2);
      voteButtons.forEach((b) => {
        expect(b).toHaveAttribute('aria-disabled', 'true');
        expect(b).toHaveAttribute('title', "Can't vote for your own comment");
      });
    });

    it('disabled for own comment', () => {
      const { container } = render(<Comment {...({ ...DefaultProps, user: null } as CommentProps)} />);

      const voteButtons = container.querySelectorAll('.comment__vote');

      expect(voteButtons).toHaveLength(2);
      voteButtons.forEach((b) => {
        expect(b).toHaveAttribute('aria-disabled', 'true');
        expect(b).toHaveAttribute('title', 'Sign in to vote');
      });
    });

    it('disabled for already upvoted comment', async () => {
      const voteSpy = jest.fn(async () => undefined);
      const { container } = render(
        <Comment
          {...(DefaultProps as CommentProps)}
          data={{ ...DefaultProps.data, vote: +1 } as CommentProps['data']}
          putCommentVote={voteSpy}
        />
      );

      const voteButtons = container.querySelectorAll('.comment__vote');

      expect(voteButtons).toHaveLength(2);

      expect(voteButtons[0]).toHaveAttribute('aria-disabled', 'true');
      fireEvent.click(voteButtons[0]);

      expect(voteSpy).not.toBeCalled();

      await waitFor(() => expect(voteButtons[1]).toHaveAttribute('aria-disabled', 'false'));
      fireEvent.click(voteButtons[1]);
      await waitFor(() => expect(voteSpy).toBeCalled());
    }, 30000);

    it('disabled for already downvoted comment', async () => {
      const voteSpy = jest.fn(async () => undefined);
      const { container } = render(
        <Comment
          {...(DefaultProps as CommentProps)}
          data={{ ...DefaultProps.data, vote: -1 } as CommentProps['data']}
          putCommentVote={voteSpy}
        />
      );

      const voteButtons = container.querySelectorAll('.comment__vote');

      expect(voteButtons).toHaveLength(2);
      expect(voteButtons[1]).toHaveAttribute('aria-disabled', 'true');
      fireEvent.click(voteButtons[1]);

      expect(voteSpy).not.toBeCalled();

      expect(voteButtons[0]).toHaveAttribute('aria-disabled', 'false');
      fireEvent.click(voteButtons[0]);
      await waitFor(() => expect(voteSpy).toBeCalled());
    }, 30000);
  });

  describe('admin controls', () => {
    it('for admin if shows admin controls', () => {
      const { container } = render(
        <Comment {...({ ...DefaultProps, user: { ...DefaultProps.user, admin: true } } as CommentProps)} />
      );

      const controls = container.querySelectorAll('.comment__control');

      expect(controls).toHaveLength(5);
      expect(controls[0]).toHaveTextContent('Copy');
      expect(controls[1]).toHaveTextContent('Pin');
      expect(controls[2]).toHaveTextContent('Hide');
      expect(controls[3]).toHaveTextContent('Block');
      expect(controls[4]).toHaveTextContent('Delete');
    });

    it('for regular user it shows only "hide"', () => {
      const { container } = render(
        <Comment {...({ ...DefaultProps, user: { ...DefaultProps.user, admin: false } } as CommentProps)} />
      );

      const controls = container.querySelectorAll('.comment__controls');
      expect(controls).toHaveLength(1);
      expect(controls[0]).toHaveTextContent('Hide');
    });

    it('verification badge clickable for admin', () => {
      const { container } = render(
        <Comment {...({ ...DefaultProps, user: { ...DefaultProps.user, admin: true } } as CommentProps)} />
      );

      expect(container.querySelector('.comment__verification')).toHaveClass('comment__verification_clickable');
    });

    it('verification badge not clickable for regular user', () => {
      const { container } = render(
        <Comment
          {...({
            ...DefaultProps,
            data: { ...DefaultProps.data, user: { ...DefaultProps.data!.user, verified: true } },
          } as CommentProps)}
        />
      );

      expect(container.querySelector('.comment__verification')).not.toHaveClass('comment__verification_clickable');
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
      };
      StaticStore.config.edit_duration = 300;
      const { container, debug } = render(<Comment {...(props as CommentProps)} />);

      expect(container.querySelector('.comment__edit-timer')).toBeInTheDocument();
    });

    it('should not be editable', () => {
      StaticStore.config.edit_duration = 300;

      const props = {
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
      };
      StaticStore.config.edit_duration = 300;

      const { container } = render(<Comment {...(props as CommentProps)} />);

      expect(container.querySelector('.comment__edit-timer')).not.toBeInTheDocument();
    });
  });
});
