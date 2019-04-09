/** @jsx h */
import { h, render } from 'preact';
import { Props, AuthPanel } from './auth-panel';
import { createDomContainer } from '../../testUtils';
import { User, PostInfo } from '../../common/types';

const DefaultProps: Partial<Props> = {
  sort: '-score',
  providers: ['google', 'github'],
  postInfo: {
    read_only: false,
    url: 'https://example.com',
    count: 3,
  },
};

describe('<AuthPanel />', () => {
  describe('For not authorized user', () => {
    let container: HTMLElement;

    createDomContainer(domContainer => {
      container = domContainer;
    });

    it('should render login form with google and github provider', () => {
      const element = <AuthPanel {...DefaultProps as Props} user={null} />;

      render(element, container);

      const authPanelColumn = container.querySelectorAll('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const authForm = authPanelColumn[0];

      expect(authForm.textContent).toEqual(expect.stringContaining('Sign in to comment using'));

      const providerLinks = authForm.querySelectorAll('.auth-panel__pseudo-link');

      expect(providerLinks[0].textContent).toEqual('Google');
      expect(providerLinks[1].textContent).toEqual('GitHub');
    });

    it('should render login form with google and github provider for read-only post', () => {
      const element = (
        <AuthPanel
          {...DefaultProps as Props}
          user={null}
          postInfo={{ ...DefaultProps.postInfo, read_only: true } as PostInfo}
        />
      );

      render(element, container);

      const authPanelColumn = container.querySelectorAll('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const authForm = authPanelColumn[0];

      expect(authForm.textContent).toEqual(expect.stringContaining('Sign in using Google or GitHub'));

      const providerLinks = authForm.querySelectorAll('.auth-panel__pseudo-link');

      expect(providerLinks[0].textContent).toEqual('Google');
      expect(providerLinks[1].textContent).toEqual('GitHub');
    });
  });
  describe('For authorized user', () => {
    let container: HTMLElement;

    createDomContainer(domContainer => {
      container = domContainer;
    });

    it('should render info about current user', () => {
      const element = <AuthPanel {...DefaultProps as Props} user={{ id: `john`, name: 'John' } as User} />;

      render(element, container);

      const authPanelColumn = container.querySelectorAll('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const userInfo = authPanelColumn[0];

      expect(userInfo.textContent).toEqual(expect.stringContaining('You signed in as John'));
    });
  });
  describe('For admin user', () => {
    let container: HTMLElement;

    createDomContainer(domContainer => {
      container = domContainer;
    });

    it('should render admin action', () => {
      const element = <AuthPanel {...DefaultProps as Props} user={{ id: `test`, admin: true, name: 'John' } as User} />;

      render(element, container);

      const adminAction = container.querySelector('.auth-panel__admin-action')!;

      expect(adminAction.textContent).toEqual('Show blocked users');
    });
  });
});
