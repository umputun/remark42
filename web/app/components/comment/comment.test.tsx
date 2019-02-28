/** @jsx h */
import { h, render } from 'preact';
import { Props, Comment } from './comment';
import { createDomContainer } from '../../testUtils';
import { User, Comment as CommentType, PostInfo } from '@app/common/types';

const DefaultProps: Partial<Props> = {
  post_info: {
    read_only: false,
  } as PostInfo,
  data: {
    text: 'test comment',
    votes: {},
    user: {
      id: 'someone',
      picture: 'somepicture-url',
    },
    locator: {
      url: 'somelocatorurl',
      site: 'remark',
    },
  } as CommentType,
  user: {
    admin: false,
    id: 'testuser',
  } as User,
};

describe('<Comment />', () => {
  describe('voting', () => {
    let container: HTMLElement;

    createDomContainer(domContainer => {
      container = domContainer;
    });

    it('disabled on user info widget', () => {
      const element = <Comment {...{ ...DefaultProps, view: 'user' } as Props} />;
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual('Voting disabled in last comments');
      }
    });

    it('disabled on read only post', () => {
      const element = (
        <Comment {...{ ...DefaultProps, post_info: { ...DefaultProps.post_info, read_only: true } } as Props} />
      );
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual("You can't vote on read-only topics");
      }
    });

    it('disabled for deleted comment', () => {
      const element = (
        // ahem
        <Comment {...{ ...DefaultProps, data: { ...DefaultProps.data, delete: true } } as Props} />
      );
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual("You can't vote for deleted comment");
      }
    });

    it('disabled for guest', () => {
      const element = (
        <Comment
          {...{
            ...DefaultProps,
            user: {
              id: 'someone',
              picture: 'somepicture-url',
            },
          } as Props}
        />
      );
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual("You can't vote for your own comment");
      }
    });

    it('disabled for own comment', () => {
      const element = <Comment {...{ ...DefaultProps, user: null } as Props} />;
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual('Only authorized users are allowed to vote');
      }
    });
  });

  describe('admin controls', () => {
    let container: HTMLElement;

    createDomContainer(domContainer => {
      container = domContainer;
    });

    it('visible for admin', () => {
      const element = <Comment {...{ ...DefaultProps, user: { ...DefaultProps.user, admin: true } } as Props} />;
      render(element, container);

      const controls = container.querySelector('.comment__controls');
      expect(controls).not.toBe(null);
    });

    it('not visible for regular user', () => {
      const element = <Comment {...{ ...DefaultProps, user: { ...DefaultProps.user, admin: false } } as Props} />;
      render(element, container);

      const controls = container.querySelector('.comment__controls');
      expect(controls).toBe(null);
    });

    it('verification badge clickable for admin', () => {
      const element = <Comment {...{ ...DefaultProps, user: { ...DefaultProps.user, admin: true } } as Props} />;
      render(element, container);

      const controls = container.querySelector('.comment__verification')!;
      expect(controls.classList.contains('comment__verification_clickable')).toBe(true);
    });

    it('verification badge not clickable for regular user', () => {
      const element = (
        <Comment
          {...{
            ...DefaultProps,
            data: { ...DefaultProps.data, user: { ...DefaultProps.data!.user, verified: true } },
          } as Props}
        />
      );
      render(element, container);

      const controls = container.querySelector('.comment__verification')!;
      expect(controls.classList.contains('comment__verification_clickable')).toBe(false);
    });
  });
});
