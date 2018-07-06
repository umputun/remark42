/** @jsx h */
import { h, render } from 'preact';
import AuthPanel from '../auth-panel';
import { createDomContainer } from 'testUtils';

describe('<AuthPanel />', () => {
  describe('For not authorized user', () => {
    let container;

    createDomContainer(({ domContainer }) => {
      container = domContainer;
    });

    it('should render login form with google and github provider', () => {
      const element = <AuthPanel user={{}} sort="-score" providers={[`google`, `github`]} />;

      render(element, container);

      const authPanelColumn = container.querySelectorAll('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const authForm = authPanelColumn[0];

      expect(authForm.textContent).toEqual(expect.stringContaining('Sign in to comment using'));

      const providerLinks = authForm.querySelectorAll('.auth-panel__pseudo-link');

      expect(providerLinks[0].textContent).toEqual('Google');
      expect(providerLinks[1].textContent).toEqual('GitHub');
    });
  });
  describe('For authorized user', () => {
    let container;

    createDomContainer(({ domContainer }) => {
      container = domContainer;
    });

    it('should render info about current user', () => {
      const element = <AuthPanel user={{ id: `test`, name: 'John' }} sort="-score" providers={[`google`, `github`]} />;

      render(element, container);

      const authPanelColumn = container.querySelectorAll('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const userInfo = authPanelColumn[0];

      expect(userInfo.textContent).toEqual(expect.stringContaining('You signed in as John. Sign out?'));
    });
  });
  describe('For admin user', () => {
    let container;

    createDomContainer(({ domContainer }) => {
      container = domContainer;
    });

    it('should render admin action', () => {
      const element = (
        <AuthPanel user={{ id: `test`, admin: true, name: 'John' }} sort="-score" providers={[`google`, `github`]} />
      );

      render(element, container);

      const adminAction = container.querySelector('.auth-panel__admin-action');

      expect(adminAction.textContent).toEqual('Show blocked');
    });
  });
});
