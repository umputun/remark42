import { h } from 'preact';
import '@testing-library/jest-dom/extend-expect';

import { render } from 'tests/utils';
import { screen } from '@testing-library/preact';

import { AuthPanel } from './auth-panel';
import { User } from 'common/types';
import { StaticStore } from 'common/static-store';

const user = {
  name: 'User Name',
  picture: 'http://localhost/picture.png',
} as User;

describe('<AuthPanel />', () => {
  describe('Unauthorised user', () => {
    it('should render as unauthorized', () => {
      render(<AuthPanel />, { user: null });

      expect(screen.queryByTestId('user-button')).not.toBeInTheDocument();
    });

    it('should NOT render settings if there IS NO hidden users', () => {
      render(<AuthPanel />, { user: null, info: { url: '', count: 10, read_only: true } });

      expect(screen.queryByTestId('settings-button')).not.toBeInTheDocument();
    });

    it('should render settings if there IS some hidden users', () => {
      render(<AuthPanel />, {
        user: null,
        info: { url: '', count: 10, read_only: true },
        hiddenUsers: { hiddenUser: user },
      });
      expect(screen.getByTestId('settings-button')).not.toBeInTheDocument();
    });
  });

  describe('Authorized user', () => {
    it('should render as authorized', () => {
      render(<AuthPanel />, { user });

      expect(screen.getByTestId('user-button')).toBeInTheDocument();
      expect(screen.getByText('User Name')).toBeInTheDocument();
      expect(screen.getByTitle('User Name')).toBeInTheDocument();
      expect(screen.getByTitle('User Name').nodeName).toBe('IMG');
      expect(screen.getByTitle('Sign Out').nodeName).toBe('BUTTON');
    });

    it('should be rendered WITH email subscription button', () => {
      StaticStore.config.email_notifications = true;

      render(<AuthPanel />, { user });
      expect(screen.getByTestId('user-notifications')).toBeInTheDocument();
    });

    it('should be rendered WITHOUT email subscription button', () => {
      StaticStore.config.email_notifications = false;

      render(<AuthPanel />, { user });
      expect(screen.getByTestId('user-notifications')).not.toBeInTheDocument();
    });
  });

  describe('Authorized administrator', () => {
    it('should render administrator action', () => {
      render(<AuthPanel />, { user: { ...user, admin: true } });

      expect(screen.getByTestId('disable-comments')).toBeInTheDocument();
    });
  });
});
