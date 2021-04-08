import '@testing-library/jest-dom';
import { h } from 'preact';
import { fireEvent, render, waitFor } from '@testing-library/preact';

import { OAuthProvider, User } from 'common/types';
import { StaticStore } from 'common/static-store';

import Auth from './auth';
import * as utils from './auth.utils';
import * as api from './auth.api';
import { getProviderData } from './components/oauth.utils';

jest.mock('react-redux', () => ({
  useDispatch: () => jest.fn(),
}));

jest.mock('hooks/useTheme', () => () => 'light');

describe('<Auth/>', () => {
  let defaultProviders = StaticStore.config.auth_providers;

  afterAll(() => {
    StaticStore.config.auth_providers = defaultProviders;
  });

  // TODO: separate tests of `useDropdown` mechanics with the hook
  describe('useDropdown', () => {
    it('should render auth with hidden dropdown', () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should close dropdown by click on button', () => {
      const { container, getByText } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(getByText('Sign In'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      fireEvent.click(getByText('Sign In'));
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should close dropdown by click outside of it', () => {
      const { container, getByText } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(getByText('Sign In'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      fireEvent.click(document);
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should close dropdown by message from parent', async () => {
      const { container, getByText } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(getByText('Sign In'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      window.postMessage('{"clickOutside": true}', '*');
      await waitFor(() => expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument());
    });
  });

  it.each([
    [[]],
    [['dev']],
    [['facebook', 'google']],
    [['facebook', 'google', 'microsoft']],
    [['facebook', 'google', 'microsoft', 'yandex']],
    [['facebook', 'google', 'microsoft', 'yandex', 'twitter']],
  ] as [OAuthProvider[]][])('should renders with %j providers', async (providers) => {
    StaticStore.config.auth_providers = providers;

    const { container, getByText, getByTitle, queryByPlaceholderText, queryByText } = render(<Auth />);

    expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

    expect(getByText('Sign In')).toHaveClass('auth-button');
    fireEvent.click(getByText('Sign In'));
    expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();
    providers.forEach((p) => {
      const { name } = getProviderData(p, 'light');
      expect(getByTitle(`Sign In with ${name}`)).toBeInTheDocument();
    });
    expect(queryByPlaceholderText('Username')).not.toBeInTheDocument();
    expect(queryByText('Submit')).not.toBeInTheDocument();
  });

  it('should render email provider', () => {
    StaticStore.config.auth_providers = ['email'];

    const { getByText, getByPlaceholderText } = render(<Auth />);

    fireEvent.click(getByText('Sign In'));
    expect(getByText('email')).toHaveClass('auth-form-title');
    expect(getByPlaceholderText('Username')).toHaveClass('auth-input-username');
    expect(getByPlaceholderText('Email Address')).toHaveClass('auth-input-email');
    expect(getByText('Submit')).toHaveClass('auth-submit');
  });

  it('should render anonymous provider', () => {
    StaticStore.config.auth_providers = ['anonymous'];

    const { getByText, getByPlaceholderText } = render(<Auth />);

    fireEvent.click(getByText('Sign In'));
    expect(getByText('anonymous')).toHaveClass('auth-form-title');
    expect(getByPlaceholderText('Username')).toHaveClass('auth-input-username');
    expect(getByText('Submit')).toHaveClass('auth-submit');
  });

  it('should render tabs with two form providers', () => {
    StaticStore.config.auth_providers = ['email', 'anonymous'];

    const { getByText, getByLabelText, getByPlaceholderText, getByDisplayValue } = render(<Auth />);

    fireEvent.click(getByText('Sign In'));
    expect(getByDisplayValue('email')).toHaveAttribute('id', 'form-provider-email');
    expect(getByText('email')).toHaveAttribute('for', 'form-provider-email');
    expect(getByText('email')).toHaveClass('auth-tabs-item');
    expect(getByDisplayValue('anonymous')).toHaveAttribute('id', 'form-provider-anonymous');
    expect(getByText('anonym')).toHaveAttribute('for', 'form-provider-anonymous');
    expect(getByText('anonym')).toHaveClass('auth-tabs-item');
    expect(getByPlaceholderText('Username')).toHaveClass('auth-input-username');
    expect(getByText('Submit')).toHaveClass('auth-submit');

    fireEvent.click(getByLabelText('email'));
    expect(getByPlaceholderText('Username')).toHaveClass('auth-input-username');
    expect(getByPlaceholderText('Email Address')).toHaveClass('auth-input-email');
    expect(getByText('Submit')).toHaveClass('auth-submit');
  });

  it('should send email and then verify forms', async () => {
    StaticStore.config.auth_providers = ['email'];
    jest.spyOn(api, 'emailSignin').mockImplementationOnce(async () => null);
    jest.spyOn(api, 'verifyEmailSignin').mockImplementationOnce(async () => ({} as User));
    jest.spyOn(utils, 'getTokenInvalidReason').mockImplementationOnce(() => null);

    const { getByText, getByPlaceholderText, getByTitle, getByRole } = render(<Auth />);

    fireEvent.click(getByText('Sign In'));
    fireEvent.change(getByPlaceholderText('Username'), { target: { value: 'username' } });
    fireEvent.change(getByPlaceholderText('Email Address'), {
      target: { value: 'email@email.com' },
    });
    fireEvent.click(getByText('Submit'));

    expect(getByRole('presentation')).toHaveClass('spinner');
    await waitFor(() => expect(api.emailSignin).toBeCalled());
    expect(api.emailSignin).toBeCalledWith('email@email.com', 'username');

    expect(getByText('Back')).toHaveClass('auth-back-button');
    expect(getByTitle('Close sign-in dropdown')).toHaveClass('auth-close-button');
    expect(getByPlaceholderText('Token')).toHaveClass('auth-token-textatea');

    fireEvent.change(getByPlaceholderText('Token'), {
      target: { value: 'token' },
    });

    fireEvent.click(getByText('Submit'));

    await waitFor(() => expect(api.verifyEmailSignin).toBeCalled());
    expect(api.verifyEmailSignin).toBeCalledWith('token');
  });

  it('should show validation error for token', async () => {
    StaticStore.config.auth_providers = ['email'];
    jest.spyOn(api, 'emailSignin').mockImplementationOnce(async () => null);

    const { getByText, getByTitle, getByPlaceholderText } = render(<Auth />);

    fireEvent.click(getByText('Sign In'));
    fireEvent.change(getByPlaceholderText('Username'), { target: { value: 'username' } });
    fireEvent.change(getByPlaceholderText('Email Address'), {
      target: { value: 'email@email.com' },
    });
    fireEvent.click(getByText('Submit'));
    await waitFor(() => expect(api.emailSignin).toBeCalled());

    expect(getByText('Back')).toHaveClass('auth-back-button');
    expect(getByTitle('Close sign-in dropdown')).toHaveClass('auth-close-button');
    expect(getByPlaceholderText('Token')).toHaveClass('auth-token-textatea');

    fireEvent.change(getByPlaceholderText('Token'), { target: { value: 'token' } });
    fireEvent.click(getByText('Submit'));
    await waitFor(() => expect(utils.getTokenInvalidReason).toBeCalled());

    expect(utils.getTokenInvalidReason).toBeCalledWith('token');

    await waitFor(() => expect(getByText('Token is invalid')).toBeInTheDocument());
    expect(getByText('Token is invalid')).toHaveClass('auth-error');
  });

  it('should send anonym form', async () => {
    StaticStore.config.auth_providers = ['anonymous'];
    jest.spyOn(api, 'anonymousSignin').mockImplementationOnce(async () => ({} as User));

    const { getByText, getByPlaceholderText, getByRole } = render(<Auth />);

    fireEvent.click(getByText('Sign In'));
    fireEvent.change(getByPlaceholderText('Username'), { target: { value: 'username' } });
    fireEvent.click(getByText('Submit'));
    expect(getByRole('presentation')).toHaveClass('spinner');
    expect(getByRole('presentation')).toHaveAttribute('aria-label', 'Loading...');
    await waitFor(() => expect(api.anonymousSignin).toBeCalled());
  });
});
