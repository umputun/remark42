/** @jsx h */
import { h, render } from 'preact';
import { Props, Comment } from './comment';
import { createDomContainer } from '../../testUtils';
import { User, Comment as CommentType, PostInfo } from '@app/common/types';
import { delay } from '@app/store/comments/utils';

const DefaultProps: Partial<Props> = {
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
      const element = <Comment {...({ ...DefaultProps, view: 'user' } as Props)} />;
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual("Voting allowed only on post's page");
      }
    });

    it('disabled on read only post', () => {
      const element = (
        <Comment {...({ ...DefaultProps, post_info: { ...DefaultProps.post_info, read_only: true } } as Props)} />
      );
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual("Can't vote on read-only topics");
      }
    });

    it('disabled for deleted comment', () => {
      const element = (
        // ahem
        <Comment {...({ ...DefaultProps, data: { ...DefaultProps.data, delete: true } } as Props)} />
      );
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual("Can't vote for deleted comment");
      }
    });

    it('disabled for guest', () => {
      const element = (
        <Comment
          {...({
            ...DefaultProps,
            user: {
              id: 'someone',
              picture: 'somepicture-url',
            },
          } as Props)}
        />
      );
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual("Can't vote for your own comment");
      }
    });

    it('disabled for own comment', () => {
      const element = <Comment {...({ ...DefaultProps, user: null } as Props)} />;
      render(element, container);

      const voteButtons = container.querySelectorAll('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      for (const b of voteButtons as any) {
        expect(b.getAttribute('aria-disabled')).toStrictEqual('true');
        expect(b.getAttribute('title')).toStrictEqual('Sign in to vote');
      }
    });

    it('disabled for already upvoted comment', async () => {
      const voteSpy = jest.fn(async () => {});
      const element = (
        <Comment
          {...(DefaultProps as Props)}
          data={{ ...DefaultProps.data, vote: +1 } as Props['data']}
          putCommentVote={voteSpy}
        />
      );
      render(element, container);

      const voteButtons = container.querySelectorAll<HTMLSpanElement>('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      expect(voteButtons[0].getAttribute('aria-disabled')).toStrictEqual('true');
      voteButtons[0].click();
      await delay(100);
      expect(voteSpy).not.toBeCalled();

      expect(voteButtons[1].getAttribute('aria-disabled')).toStrictEqual('false');
      voteButtons[1].click();
      await delay(100);
      expect(voteSpy).toBeCalled();
    }, 30000);

    it('disabled for already downvoted comment', async () => {
      const voteSpy = jest.fn(async () => {});
      const element = (
        <Comment
          {...(DefaultProps as Props)}
          data={{ ...DefaultProps.data, vote: -1 } as Props['data']}
          putCommentVote={voteSpy}
        />
      );
      render(element, container);

      const voteButtons = container.querySelectorAll<HTMLSpanElement>('.comment__vote');
      expect(voteButtons.length).toStrictEqual(2);

      expect(voteButtons[1].getAttribute('aria-disabled')).toStrictEqual('true');
      voteButtons[1].click();
      await delay(100);
      expect(voteSpy).not.toBeCalled();

      expect(voteButtons[0].getAttribute('aria-disabled')).toStrictEqual('false');
      voteButtons[0].click();
      await delay(100);
      expect(voteSpy).toBeCalled();
    }, 30000);
  });

  describe('admin controls', () => {
    let container: HTMLElement;

    createDomContainer(domContainer => {
      container = domContainer;
    });

    it('for admin if shows admin controls', () => {
      const element = <Comment {...({ ...DefaultProps, user: { ...DefaultProps.user, admin: true } } as Props)} />;
      render(element, container);

      const controls = container.querySelectorAll('.comment__controls > span');
      expect(controls!.length).toBe(5);
      expect(controls![0].textContent).toBe('Copy');
      expect(controls![1].textContent).toBe('Pin');
      expect(controls![2].textContent).toBe('Hide');
      expect(controls![3].childNodes[0].textContent).toBe('Block');
      expect(controls![4].textContent).toBe('Delete');
    });

    it('for regular user it shows only "hide"', () => {
      const element = <Comment {...({ ...DefaultProps, user: { ...DefaultProps.user, admin: false } } as Props)} />;
      render(element, container);

      const controls = container.querySelectorAll('.comment__controls > span');
      expect(controls!.length).toBe(1);
      expect(controls![0].textContent).toBe('Hide');
    });

    it('verification badge clickable for admin', () => {
      const element = <Comment {...({ ...DefaultProps, user: { ...DefaultProps.user, admin: true } } as Props)} />;
      render(element, container);

      const controls = container.querySelector('.comment__verification')!;
      expect(controls.classList.contains('comment__verification_clickable')).toBe(true);
    });

    it('verification badge not clickable for regular user', () => {
      const element = (
        <Comment
          {...({
            ...DefaultProps,
            data: { ...DefaultProps.data, user: { ...DefaultProps.data!.user, verified: true } },
          } as Props)}
        />
      );
      render(element, container);

      const controls = container.querySelector('.comment__verification')!;
      expect(controls.classList.contains('comment__verification_clickable')).toBe(false);
    });
  });
});
